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
package app

import (
	"fmt"
	"log/slog"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"fox-audio/audio"
	"fox-audio/display"
	"fox-audio/model"
	"fox-audio/reaper"
	"fox-audio/shared"
)

var (
	displayHandle display.UI
	audioServer   *audio.JackServer
	ports         []*audio.Port
	outputFiles   []*audio.OutputFile
)

func ConfigureTextLogger() {
	// text logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.Level(slog.LevelDebug),
	}))
	slog.SetDefault(logger)
}

func ConfigureFileLogger() {
	f, err := os.Create("/Users/flip/projects/personal/fox-recorder/fox.log")

	if err != nil {
		panic(err)
	}

	handler := slog.NewTextHandler(f, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	shared.HijackLogging()
	shared.EnableSlogLogging()
}

func ConfigureUiLogger() {
	handler := shared.NewTuiLogHandler(displayHandle, slog.LevelDebug, func(message string) {
		displayHandle.IncrementErrorCount()
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	shared.HijackLogging()
	shared.EnableSlogLogging()
}

func runEngine(config *model.Config, profile *model.Profile) {
	if config.OutputType == model.OutputTUI {
		displayHandle = display.NewTui()
	} else if config.OutputType == model.OutputJSON {
		displayHandle = display.NewJsonUI(os.Stdout)
	}

	reaper.SetPanicHandler(displayHandle.HandlePanic)

	displayHandle.Initalize()
	displayHandle.SetTransportStatus(display.StatusStarting)
	displayHandle.Start()
	reaper.Callback("tui", displayHandle.Shutdown)

	statsShutdownChan := initStatistics(profile)
	reaper.Callback("stats", func() { statsShutdownChan <- true })

	// TODO: wait for user confirmation on error
	// TODO: make this show a session summary instead
	reaper.Callback("wait", func() { time.Sleep(3 * time.Second) })

	shared.CatchSigint(func() {
		slog.Info("Caught sigint, calling reaper")
		reaper.Reap()
	})

	// ConfigureTextLogger()
	ConfigureUiLogger()
	// ConfigureFileLogger()

	if !config.SimulationOptions.EnableSimulation {
		audioServer = audio.NewServer(config, profile, jackInfo, jackError)

		jackRunning := getJackServer(profile)

		if jackRunning {
			displayHandle.SetTransportStatus(display.StatusPaused)

			connected := audioServer.Connect(jackProcess, jackXrun, jackShutdown)

			if !connected {
				slog.Error("Failed to connect to JACK server")
				reaper.Reap()
			} else {
				reaper.Callback("disconnect jack server", audioServer.Disconnect)

				setupCycleBuffer(profile)

				// only register input ports, for now
				audioServer.RegisterPorts(true, false)

				audioServer.PrepareOutputFiles()
				outputFiles = audioServer.GetOutputFiles()
				uiSetupOutputFiles()

				ports = audioServer.GetInputPorts()
				uiSetupLevelMeters()

				audioServer.ActivateClient()

				startDiskWriter(profile)

				audioServer.ConnectPorts(true, false)

				uiSetOuputFormat(profile)

				transportRecord = true
			}
		}
	}

	// this blocks until the jack connection shuts down
	if config.SimulationOptions.EnableSimulation {
		displayHandle.SetChannelCount(config.SimulationOptions.ChannelCount)
		startSimulation(config.SimulationOptions)
	}

	reaper.Callback("shutdown status", doShutdown)

	// wait for everything to finalize and shutdown
	reaper.Wait()
}

func doShutdown() {
	transportRecord = false
	displayHandle.SetTransportStatus(display.StatusShuttingDown)
}

func uiSetupOutputFiles() {
	uiOutputFiles := make([]model.UiOutputFile, len(outputFiles))
	for ofIndex, outputFile := range outputFiles {

		ports := make([]string, len(outputFile.InputPorts))
		for pIndex, port := range outputFile.InputPorts {
			ports[pIndex] = strings.Split(port.GetJackPort().GetName(), "_")[1]
		}

		uiOutputFiles[ofIndex] = model.UiOutputFile{
			Ports: ports,
			Name:  outputFile.ChannelName,
			Size:  0,
		}
	}
	displayHandle.SetOutputFiles(uiOutputFiles)
}

func uiSetupLevelMeters() {
	displayHandle.SetChannelCount(len(ports))
	signalLevels = make([]model.SignalLevel, len(ports))

	for i, port := range ports {
		displayHandle.SetChannelArmStatus(i, port.IsArmed())
	}
}

func uiSetOuputFormat(profile *model.Profile) {
	sampleRateStr := strconv.FormatFloat(float64(audioServer.GetSampleRate())/1000.0, 'f', -1, 64)
	displayHandle.SetAudioFormat(fmt.Sprintf("%d bit / %s KHz", profile.Output.BitDepth, sampleRateStr))
	displayHandle.SetTransportStatus(display.StatusRecording)
	displayHandle.SetProfileName(profile.Name)
	displayHandle.SetTakeName(profile.Output.Take)
	displayHandle.SetDirectory(profile.Output.Directory)
}

func setupCycleBuffer(profile *model.Profile) {
	// set cycle buffer size .. this needs to be proportional to the disk
	// buffer so that the buffer has time to fill before the jack processor
	// deadlocks waiting for data to be processed
	cycleBufferSize := int(math.Ceil(float64(audioServer.GetSampleRate())*float64(int(profile.Output.BufferSizeSeconds))) / float64(audioServer.GetFramesPerPeriod()))
	cycleDoneChannel = make(chan bool, cycleBufferSize*5)
}

func getJackServer(profile *model.Profile) bool {
	jackRunning := true

	if profile.AudioServer.AutoStart {
		err := audioServer.StartServer()
		if err != nil {
			slog.Error(err.Error())
			displayHandle.SetTransportStatus(display.StatusFailed)
			jackRunning = false
			reaper.Reap()
		} else {
			reaper.Callback("stop jack server", audioServer.StopServer)
		}
	}

	return jackRunning
}
