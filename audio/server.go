// =================================================================================
//
//			fox-audio - https://www.foxhollow.cc/projects/fox-audio/
//
//		 Fox Audio is a simple CLI utility for recording and playback of
//	  multitrack audio straight to disk by utilizing the JACK audio server
//
//		 Copyright (c) 2024 Steve Cross <flip@foxhollow.cc>
//
//			Licensed under the Apache License, Version 2.0 (the "License");
//			you may not use this file except in compliance with the License.
//			You may obtain a copy of the License at
//
//			     http://www.apache.org/licenses/LICENSE-2.0
//
//			Unless required by applicable law or agreed to in writing, software
//			distributed under the License is distributed on an "AS IS" BASIS,
//			WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//			See the License for the specific language governing permissions and
//			limitations under the License.
//
// =================================================================================
package audio

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"fox-audio/model"
	"fox-audio/reaper"
	"fox-audio/util"

	"github.com/go-audio/wav"
	"github.com/hairlesshobo/go-jack"
)

type JackServer struct {
	profile *model.Profile

	clientName      string
	audioInterface  string
	driver          string
	device          string
	sampleRate      int
	framesPerPeriod int

	outputDirectory string
	take            string

	ports       []*Port
	outputFiles []*OutputFile

	jackClient *jack.Client

	clientConnected bool
	shutdownMutex   sync.Mutex

	cmd *exec.Cmd
}

func NewServer(clientName string, profile *model.Profile) *JackServer {
	audioInterfaceParts := strings.Split(profile.AudioServer.Interface[0], "/")

	// TODO: add support for multiple interfaces
	server := JackServer{
		profile: profile,

		clientName:      clientName,
		audioInterface:  profile.AudioServer.Interface[0],
		sampleRate:      profile.AudioServer.SampleRate,
		framesPerPeriod: profile.AudioServer.FramesPerPeriod,

		driver: audioInterfaceParts[0],
		device: audioInterfaceParts[1],

		clientConnected: false,
		shutdownMutex:   sync.Mutex{},

		ports: make([]*Port, 0),
	}

	return &server
}

func (server *JackServer) StartServer() {
	ready := make(chan bool)

	go func() {
		reaper.Register("jack server")

		slog.Info("Starting JACK server...")
		// TODO: dynamically find jackd binary
		// TODO: allow to specify jack binary in config
		// /usr/local/bin/jackd -dcoreaudio -d'AppleUSBAudioEngine:BEHRINGER:X-USB:42D1635E:1,2' -r48000 -p4096 -C
		server.cmd = exec.Command("/usr/local/bin/jackd")

		// TODO: add this to config
		// server.cmd.Args = append(server.cmd.Args, "-v")
		server.cmd.Args = append(server.cmd.Args, fmt.Sprintf("-d%s", server.driver))

		if server.device != "" {
			server.cmd.Args = append(server.cmd.Args, fmt.Sprintf("-d%s", server.device))
		}

		server.cmd.Args = append(server.cmd.Args, fmt.Sprintf("-r%d", server.sampleRate))
		server.cmd.Args = append(server.cmd.Args, fmt.Sprintf("-p%d", server.framesPerPeriod))

		stdout, err := server.cmd.StdoutPipe()

		// TODO: handle jack startup failure!!

		if err != nil {
			slog.Error("Error occurred running 'diskutil activity' command: " + err.Error())
			return
		}

		if err = server.cmd.Start(); err != nil {
			slog.Error("Error occurred starting 'diskutil activity' command: " + err.Error())
			return
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			// not using reaper.Reaped() here because this should end on its own once the jack server is killed
			line := scanner.Text()

			slog.Debug("jackd: " + line)

			// found input channel
			if strings.Contains(line, "JACK input port =") {
				parts := strings.Split(line, " ==> ")         // Input channel = 0 ==> JACK input port = 0
				inputDescriptor := parts[1]                   // JACK input port = 0
				parts = strings.Split(inputDescriptor, " = ") // 0
				inputNum, _ := strconv.Atoi(strings.Trim(parts[1], ""))
				inputNum++

				server.ports = append(server.ports, newPort(In, fmt.Sprintf("in_%d", inputNum), fmt.Sprintf("system:capture_%d", inputNum)))
			} else if strings.Contains(line, "JACK output port =") {
				parts := strings.Split(line, " ==> ")          // JACK output port = 1 ==> output channel = 1
				outputDescriptor := parts[0]                   // JACK output port = 1
				parts = strings.Split(outputDescriptor, " = ") // 1
				outputNum, _ := strconv.Atoi(strings.Trim(parts[1], ""))
				outputNum++

				server.ports = append(server.ports, newPort(Out, fmt.Sprintf("out_%d", outputNum), fmt.Sprintf("system:playback_%d", outputNum)))
			} else if strings.Contains(line, "driver is running...") {
				ready <- true
			}
		}

		reaper.Done("jack server")
	}()

	<-ready
}

func (server *JackServer) StopServer() {
	if server != nil {
		server.Disconnect()

		// TODO: make sure process is running?
		server.cmd.Process.Kill()
		server.cmd.Wait()
	}
}

func (server *JackServer) Connect() {
	reaper.Register("jack client")

	slog.Info("Connecting to JACK server")

	var jackStatus int
	server.jackClient, jackStatus = jack.ClientOpen(server.clientName, jack.NoStartServer)

	if jackStatus != 0 {
		slog.Error(fmt.Sprintf("JACK Status: %s", jack.StrError(jackStatus)))
		return
	}

	server.clientConnected = true

	slog.Info("JACK server connected")
}

func (server *JackServer) Disconnect() {
	server.shutdownMutex.Lock()

	if server.clientConnected {
		// disconnect all ports
		server.DisconnectAllPorts()

		// deactivate
		// server.DeactivateClient()

		slog.Info("Allowing jack to finish processing")
		time.Sleep(1 * time.Second)

		slog.Info("Disconnecting JACK client")

		jackStatus := server.jackClient.Close()

		if jackStatus != 0 {
			slog.Error(fmt.Sprintf("JACK Status: %s", jack.StrError(jackStatus)))
			return
		}

		server.clientConnected = false
		slog.Info("JACK client disconnected")
		reaper.Done("jack client")
	} else {
		slog.Warn("JACK client already disconnected")
	}

	server.shutdownMutex.Unlock()
}

func (server *JackServer) DisconnectAllPorts() {
	slog.Info("Disconnecting all audio ports")

	for _, port := range server.GetAllPorts() {
		if !port.connected {
			continue
		}

		var inName string
		var outName string

		if port.portDirection == In {
			inName = port.jackName
			outName = fmt.Sprintf("%s:%s", server.clientName, port.myName)
		} else if port.portDirection == Out {
			inName = fmt.Sprintf("%s:%s", server.clientName, port.myName)
			outName = port.jackName
		}

		slog.Debug(fmt.Sprintf("Disconnected port %s from port %s", inName, outName))
		server.jackClient.Disconnect(inName, outName)
		port.connected = false
	}
}

func (server *JackServer) GetAllPorts() []*Port {
	return server.ports
}

func (server *JackServer) findJackPort(name string) *Port {
	for _, port := range server.ports {
		if port.jackName == name {
			return port
		}
	}

	return nil
}

func (server *JackServer) GetOutputFiles() []*OutputFile {
	return server.outputFiles
}

func (server *JackServer) prepareOutputDirectory() {
	outputDir, err := util.ResolveHomeDirPath(time.Now().Format(server.profile.Output.Directory))
	if err != nil {
		slog.Error("Failed to resolve home user dir: " + err.Error())
		reaper.Reap()
		return
	}

	if !util.DirectoryExists(outputDir) {
		slog.Info("Creating output directory: " + outputDir)
		os.MkdirAll(outputDir, 0755)
	}

	server.outputDirectory = outputDir
	server.take = util.GetTake(server.outputDirectory)
}

func (server *JackServer) PrepareOutputFiles() {
	server.prepareOutputDirectory()

	for _, channel := range server.profile.Channels {
		portNumbers := make([]string, len(channel.Ports))

		for i, channel := range channel.Ports {
			portNumbers[i] = fmt.Sprintf("%02d", channel)
		}

		fileName := fmt.Sprintf("%s_channel%s_%s.wav", server.take, strings.Join(portNumbers, "-"), channel.ChannelName)

		outputFile := OutputFile{
			FileName:     fileName,
			FilePath:     path.Join(server.outputDirectory, fileName),
			InputPorts:   make([]*Port, len(channel.Ports)),
			ChannelCount: len(channel.Ports),
			BitDepth:     server.profile.Output.BitDepth,
			SampleRate:   server.profile.AudioServer.SampleRate,
			FileOpen:     false,
		}

		slog.Info("Creating output file " + outputFile.FilePath)

		var err error
		outputFile.FileHandle, err = os.Create(outputFile.FilePath)
		if err != nil {
			slog.Error("error creating %s: %s", outputFile.FilePath, err)
		}

		outputFile.Encoder = wav.NewEncoder(outputFile.FileHandle, outputFile.SampleRate, outputFile.BitDepth, len(channel.Ports), 1)

		for channelNum, channelPort := range channel.Ports {
			jackPort := server.findJackPort(fmt.Sprintf("system:capture_%d", channelPort))

			if jackPort != nil {
				outputFile.InputPorts[channelNum] = jackPort

				success := jackPort.AllocateBuffer(int(float64(server.profile.AudioServer.SampleRate) * server.profile.AudioServer.BufferSizeSeconds))

				// TODO: make sure a port can only be assigned once - add channel to port and compare that its not a different channel
				if !success {
					slog.Error("Failed to allocate buffer for port " + jackPort.jackName)
				}
			}
		}

		outputFile.FileOpen = true

		server.outputFiles = append(server.outputFiles, &outputFile)
	}
}

func (server *JackServer) GetPorts(direction PortDirection) []*Port {
	ports := make([]*Port, 0)

	for _, port := range server.GetAllPorts() {
		if port.portDirection == direction {
			ports = append(ports, port)
		}
	}

	return ports
}

func (server *JackServer) GetInputPorts() []*Port {
	return server.GetPorts(In)
}

func (server *JackServer) GetOutputPorts() []*Port {
	return server.GetPorts(Out)
}

func (server *JackServer) RegisterPorts(registerInput bool, registerOutput bool) {
	slog.Info("Registering audio ports...")

	for _, port := range server.GetAllPorts() {
		var jackDirection uint64

		if port.portDirection == In {
			if !registerInput {
				continue
			}

			jackDirection = jack.PortIsInput
		}

		if port.portDirection == Out {
			if !registerOutput {
				continue
			}

			jackDirection = jack.PortIsOutput
		}

		slog.Debug("Registered port " + port.myName)
		jackPort := server.jackClient.PortRegister(port.myName, jack.DEFAULT_AUDIO_TYPE, jackDirection, 0)
		port.setJackPort(jackPort)
	}
}

func (server *JackServer) SetProcessCallback(callback func(nframes uint32) int) {
	if code := server.jackClient.SetProcessCallback(callback); code != 0 {
		slog.Error(fmt.Sprintf("Failed to set process callback: %s", jack.StrError(code)))
		return
	}
}

func (server *JackServer) SetErrorCallback(callback func(string)) {
	jack.SetErrorFunction(callback)
}

func (server *JackServer) SetInfoCallback(callback func(string)) {
	jack.SetInfoFunction(callback)
}

func (server *JackServer) SetShutdownCallback(callback func()) {
	server.jackClient.OnShutdown(callback)
}

func (server *JackServer) SetXrunCallback(callback func() int) {
	server.jackClient.SetXRunCallback(callback)
}

func (server *JackServer) GetSampleRate() uint32 {
	return server.jackClient.GetSampleRate()
}

func (server *JackServer) CloseOutputFiles() {
	for _, outputFile := range server.outputFiles {
		outputFile.Close()
	}
}

func (server *JackServer) ActivateClient() {
	slog.Info("Activating jack client")

	// activate client
	if code := server.jackClient.Activate(); code != 0 {
		slog.Error(fmt.Sprintf("Failed to activate client: %s", jack.StrError(code)))
		return
	}
}

// func (server *JackServer) DeactivateClient() {
// 	slog.Info("Deactivating jack client")

// 	// deactivate client
// 	if code := server.jackClient.Deactivate(); code != 0 {
// 		slog.Error(fmt.Sprintf("Failed to deactivate client: %s", jack.StrError(code)))
// 		return
// 	}
// }

func (server *JackServer) ConnectPorts(connectInput bool, connectOutput bool) {
	slog.Info("Connecting audio ports")

	for _, port := range server.GetAllPorts() {
		var inName string
		var outName string

		if port.portDirection == In {
			if !connectInput {
				continue
			}

			inName = port.jackName
			outName = fmt.Sprintf("%s:%s", server.clientName, port.myName)
		} else if port.portDirection == Out {
			if !connectOutput {
				continue
			}

			inName = fmt.Sprintf("%s:%s", server.clientName, port.myName)
			outName = port.jackName
		}

		slog.Debug(fmt.Sprintf("Connected port %s to port %s", inName, outName))
		server.jackClient.Connect(inName, outName)
		port.connected = true
	}

	slog.Info("Audio ports connected")
}
