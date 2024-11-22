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
	"os/exec"
	"strconv"
	"strings"

	"github.com/hairlesshobo/go-jack"
)

type JackServer struct {
	clientName       string
	audioInterface   string
	driver           string
	device           string
	sampleRate       int
	samplesPerPeriod int

	ports []*Port

	jackClient *jack.Client

	cmd *exec.Cmd
}

func NewServer(clientName string, audioInterface string, sampleRate int, samplesPerPeriod int) *JackServer {
	audioInterfaceParts := strings.Split(audioInterface, "/")

	server := JackServer{
		clientName:       clientName,
		audioInterface:   audioInterface,
		sampleRate:       sampleRate,
		samplesPerPeriod: samplesPerPeriod,

		driver: audioInterfaceParts[0],
		device: audioInterfaceParts[1],

		ports: make([]*Port, 0),
	}

	return &server
}

func (server *JackServer) StartServer() {
	// TODO: spawn as goroutine, add channel to wait for server to start then return input and output ports
	ready := make(chan bool)

	go func() {
		slog.Info("Starting JACK server...")
		// /usr/local/bin/jackd -dcoreaudio -d'AppleUSBAudioEngine:BEHRINGER:X-USB:42D1635E:1,2' -r48000 -p4096 -C
		server.cmd = exec.Command(
			"/usr/local/bin/jackd",
			"-v",
			fmt.Sprintf("-d%s", server.driver),
			fmt.Sprintf("-d%s", server.device),
			fmt.Sprintf("-r%d", server.sampleRate),
			fmt.Sprintf("-p%d", server.samplesPerPeriod),
		)
		stdout, err := server.cmd.StdoutPipe()

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
			line := scanner.Text()

			slog.Debug(line)

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
	}()

	<-ready
}

func (server *JackServer) StopServer() {
	if server != nil {
		if server.jackClient != nil {
			server.jackClient.Close()
		}

		// TODO: make sure process is running?
		server.cmd.Process.Kill()
		server.cmd.Wait()
	}
}

func (server *JackServer) Connect() {
	slog.Info("Connecting to JACK server")

	var jackStatus int
	server.jackClient, jackStatus = jack.ClientOpen(server.clientName, jack.NoStartServer)

	if jackStatus != 0 {
		slog.Error(fmt.Sprintf("JACK Status: %s", jack.StrError(jackStatus)))
		return
	}

	slog.Info("JACK server connected")
}

func (server *JackServer) GetAllPorts() []*Port {
	return server.ports
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

func (server *JackServer) ActivateClient() {
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
}

// func prepareCommand(command string, arg ...string) (string, []string) {
// 	// I'm sure this function isn't perfect, but it should simplify things for me a bit
// 	// so its what i am going to go with
// 	commandParts := strings.Split(command, " ")

// 	if len(commandParts) <= 0 {
// 		panic("No command was passed")
// 	}

// 	commandName := commandParts[0]
// 	var commandArgs []string

// 	for i := 1; i < len(commandParts); i++ {
// 		part := commandParts[i]

// 		if strings.HasPrefix(part, "%s") {
// 			// we found a placeholder
// 			wants_index, err := strconv.Atoi(strings.TrimPrefix(part, "%s"))
// 			if err != nil {
// 				panic("Error parsing index placeholder")
// 			}

// 			if len(arg) > wants_index {
// 				commandArgs = append(commandArgs, arg[wants_index])
// 			} else {
// 				panic("Invalid placeholder index provided")
// 			}
// 		} else {
// 			// no placeholder
// 			commandArgs = append(commandArgs, part)
// 		}
// 	}

// 	return commandName, commandArgs
// }

// func callExternalCommand(command string, arg ...string) (string, int, error) {
// 	cmdName, cmdArgs := prepareCommand(command, arg...)
// 	// slog.Debug("Calling external command", slog.String("command", cmdName), slog.Any("args", cmdArgs))

// 	cmd := exec.Command(cmdName, cmdArgs...)
// 	output, err := cmd.Output()
// 	if err != nil {
// 		if exiterr, ok := err.(*exec.ExitError); ok {
// 			// slog.Debug(fmt.Sprintf("Exit Status: %d", exiterr.ExitCode()))
// 			// slog.Debug(fmt.Sprintf("stderr output: %s", string(exiterr.Stderr)))
// 			return string(exiterr.Stderr), exiterr.ExitCode(), err
// 		} else {
// 			// slog.Warn(fmt.Sprintf("Error occurred while calling '%s' command: %s", cmdName, err.Error()))
// 			return "", -666, err
// 		}
// 	}

// 	return string(output), 0, nil
// }
