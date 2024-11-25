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
	"strings"
	"sync"
	"time"

	"fox-audio/model"
	"fox-audio/reaper"

	"github.com/go-audio/wav"
	"github.com/hairlesshobo/go-jack"
)

type JackServer struct {
	config  *model.Config
	profile *model.Profile

	audioInterface  string
	driver          string
	device          string
	sampleRate      int
	framesPerPeriod int

	ports       []*Port
	outputFiles []*OutputFile

	jackClient *jack.Client

	clientConnected bool
	shutdownMutex   sync.Mutex

	cmd *exec.Cmd
}

func NewServer(config *model.Config, profile *model.Profile) *JackServer {
	audioInterfaceParts := strings.Split(profile.AudioServer.Interface[0], "/")

	// TODO: add support for trying multiple audio interfaces
	server := JackServer{
		config:  config,
		profile: profile,

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
		// TODO: allow to specify jack binary in config
		// TODO: dynamically find jackd binary
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
		// TODO: handle jack stderr

		if err != nil {
			slog.Error("Error occurred running 'jackd' command: " + err.Error())
			return
		}

		if err = server.cmd.Start(); err != nil {
			slog.Error("Error occurred starting 'jackd' command: " + err.Error())
			return
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			// not using reaper.Reaped() here because this should end on its own once the jack server is killed
			line := scanner.Text()

			slog.Debug("jackd: " + line)

			if strings.Contains(line, "driver is running...") {
				ready <- true
			}
		}

		reaper.Done("jack server")
	}()

	<-ready
}

func (server *JackServer) Connect() {
	reaper.Register("jack client")

	slog.Info("Connecting to JACK server")

	var jackStatus int
	server.jackClient, jackStatus = jack.ClientOpen(server.config.JackClientName, jack.NoStartServer)

	if jackStatus != 0 {
		slog.Error(fmt.Sprintf("JACK Status: %s", jack.StrError(jackStatus)))
		return
	}

	server.clientConnected = true

	slog.Info("JACK server connected")

	server.populatePorts()
}

func (server *JackServer) PrepareOutputFiles() {
	for _, channel := range server.profile.Channels {
		portNumbers := make([]string, len(channel.Ports))

		for i, channel := range channel.Ports {
			portNumbers[i] = fmt.Sprintf("%02d", channel)
		}

		fileName := fmt.Sprintf("%s_channel%s_%s.wav", server.profile.Output.Take, strings.Join(portNumbers, "-"), channel.ChannelName)

		outputFile := &OutputFile{
			ChannelName:  channel.ChannelName,
			Enabled:      !channel.Disabled,
			FileName:     fileName,
			FilePath:     path.Join(server.profile.Output.Directory, fileName),
			InputPorts:   make([]*Port, len(channel.Ports)),
			ChannelCount: len(channel.Ports),
			BitDepth:     server.profile.Output.BitDepth,
			SampleRate:   server.profile.AudioServer.SampleRate,
			FileOpen:     false,
		}

		// if the channel isn't enabled, we skip creating output files or buffers
		if !channel.Disabled {
			slog.Info("Creating output file " + outputFile.FilePath)

			var err error
			outputFile.FileHandle, err = os.Create(outputFile.FilePath)
			if err != nil {
				slog.Error("error creating %s: %s", outputFile.FilePath, err)
			}

			outputFile.Encoder = wav.NewEncoder(outputFile.FileHandle, outputFile.SampleRate, outputFile.BitDepth, len(channel.Ports), 1)

			for channelNum, channelPort := range channel.Ports {
				jackPort := server.findJackPort(fmt.Sprintf("%d", channelPort), In)

				if jackPort != nil {
					// this should make sure a port can only be assigned once
					if jackPort.outputFile != nil && jackPort.outputFile != outputFile {
						slog.Error(fmt.Sprintf("Error assigning output port to file '%s' because input port %d is already assigned to '%s'", channel.ChannelName, channelPort, jackPort.outputFile.ChannelName))
						reaper.Reap()
						return
					}

					jackPort.outputFile = outputFile
					outputFile.InputPorts[channelNum] = jackPort

					success := jackPort.AllocateBuffer(int(float64(server.profile.AudioServer.SampleRate) * server.profile.Output.BufferSizeSeconds))

					if !success {
						slog.Error("Failed to allocate buffer for port " + jackPort.jackName)
						reaper.Reap()
						return
					}
				}
			}

			outputFile.FileOpen = true
		}

		server.outputFiles = append(server.outputFiles, outputFile)
	}
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

func (server *JackServer) ActivateClient() {
	slog.Info("Activating jack client")

	// activate client
	if code := server.jackClient.Activate(); code != 0 {
		slog.Error(fmt.Sprintf("Failed to activate client: %s", jack.StrError(code)))
		return
	}
}

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
			outName = fmt.Sprintf("%s:%s", server.config.JackClientName, port.myName)
		} else if port.portDirection == Out {
			if !connectOutput {
				continue
			}

			inName = fmt.Sprintf("%s:%s", server.config.JackClientName, port.myName)
			outName = port.jackName
		}

		slog.Debug(fmt.Sprintf("Connected port %s to port %s", inName, outName))
		server.jackClient.Connect(inName, outName)
		port.connected = true
	}

	slog.Info("Audio ports connected")
}

func (server *JackServer) CloseOutputFiles() {
	for _, outputFile := range server.outputFiles {
		outputFile.Close()
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
			outName = fmt.Sprintf("%s:%s", server.config.JackClientName, port.myName)
		} else if port.portDirection == Out {
			inName = fmt.Sprintf("%s:%s", server.config.JackClientName, port.myName)
			outName = port.jackName
		}

		slog.Debug(fmt.Sprintf("Disconnected port %s from port %s", inName, outName))
		server.jackClient.Disconnect(inName, outName)
		port.connected = false
	}
}

func (server *JackServer) Disconnect() {
	server.shutdownMutex.Lock()

	if server.clientConnected {
		// disconnect all ports
		server.DisconnectAllPorts()

		// important: this seems to break everything.. shrug.
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

func (server *JackServer) StopServer() {
	if server != nil {
		server.Disconnect()

		// TODO: make sure process is running? add boolean to server struct and set/clear it in StartServer() w/ mutex
		server.cmd.Process.Kill()
		server.cmd.Wait()
	}
}

//
// getter functions
//

func (server *JackServer) GetFramesPerPeriod() int {
	return int(server.jackClient.GetBufferSize())
}

func (server *JackServer) GetSampleRate() int {
	return int(server.jackClient.GetSampleRate())
}

func (server *JackServer) GetAllPorts() []*Port {
	return server.ports
}

func (server *JackServer) GetOutputFiles() []*OutputFile {
	return server.outputFiles
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

//
// callback registration
//

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

//
// private functions
//

func (server *JackServer) findJackPort(name string, portDirection PortDirection) *Port {
	for _, port := range server.ports {
		if port.portDirection != portDirection {
			continue
		}

		if strings.HasSuffix(port.jackName, name) {
			return port
		}
	}

	return nil
}

func (server *JackServer) populatePorts() {
	// get input ports
	inputPorts := server.jackClient.GetPorts(server.config.HardwarePortConnectionPrefix+":out*", "", jack.PortIsOutput) // | jack.PortIsPhysical)
	for i, port := range inputPorts {
		server.ports = append(server.ports, newPort(In, fmt.Sprintf("in_%d", i+1), port))
	}

	// get output ports
	// outputPorts := server.jackClient.GetPorts("", "", jack.PortIsInput|jack.PortIsPhysical)
	// for i, port := range outputPorts {
	// 	// server.ports = append(server.ports, newPort(Out, fmt.Sprintf("out_%d", outputNum), fmt.Sprintf("system:playback_%d", outputNum)))
	// 	server.ports = append(server.ports, newPort(In, fmt.Sprintf("out_%d", i+1), port))
	// }
}
