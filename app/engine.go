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

type displayObj struct {
	tui *display.Tui
}

var (
	displayHandle displayObj
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

func ConfigureTuiLogger() {
	handler := shared.NewTuiLogHandler(displayHandle.tui, slog.LevelDebug, func(message string) {
		displayHandle.tui.IncrementErrorCount()
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	shared.HijackLogging()
	shared.EnableSlogLogging()
}

func runEngine(config *model.Config, profile *model.Profile, simulationOptions *model.SimulationOptions) {
	displayHandle.tui = display.NewTui()
	displayHandle.tui.Initalize()
	displayHandle.tui.SetTransportStatus(display.StatusStarting)
	displayHandle.tui.Start()
	reaper.Callback("tui", displayHandle.tui.Shutdown)

	statsShutdownChan := initStatistics(profile)
	reaper.Callback("stats", func() { statsShutdownChan <- true })

	// TODO: wait for user confirmation on error
	reaper.Callback("wait", func() { time.Sleep(6 * time.Second) })

	shared.CatchSigint(func() {
		slog.Info("Caught sigint, calling reaper")
		reaper.Reap()
	})

	// ConfigureTextLogger()
	ConfigureTuiLogger()
	// ConfigureFileLogger()

	if !simulationOptions.EnableSimulation {
		audioServer = audio.NewServer(config, profile, jackInfo, jackError)

		jackRunning := getJackServer(profile)

		if jackRunning {
			displayHandle.tui.SetTransportStatus(display.StatusPaused)

			audioServer.Connect(jackProcess, jackXrun, jackShutdown)
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

	// this blocks until the jack connection shuts down
	if simulationOptions.EnableSimulation {
		displayHandle.tui.SetChannelCount(simulationOptions.ChannelCount)
		startSimulation(simulationOptions)
	}

	reaper.Callback("shutdown status", doShutdown)

	// wait for everything to finalize and shutdown
	reaper.Wait()
}

func doShutdown() {
	transportRecord = false
	displayHandle.tui.SetTransportStatus(display.StatusShuttingDown)
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
	displayHandle.tui.SetOutputFiles(uiOutputFiles)
}

func uiSetupLevelMeters() {
	displayHandle.tui.SetChannelCount(len(ports))
	signalLevels = make([]*model.SignalLevel, len(ports))

	for i, port := range ports {
		displayHandle.tui.SetChannelArmStatus(i, port.IsArmed())
	}
}

func uiSetOuputFormat(profile *model.Profile) {
	sampleRateStr := strconv.FormatFloat(float64(audioServer.GetSampleRate())/1000.0, 'f', -1, 64)
	displayHandle.tui.SetAudioFormat(fmt.Sprintf("%d bit / %s KHz", profile.Output.BitDepth, sampleRateStr))
	displayHandle.tui.SetTransportStatus(display.StatusRecording)
	displayHandle.tui.SetProfileName(profile.Name)
	displayHandle.tui.SetTakeName(profile.Output.Take)
	displayHandle.tui.SetDirectory(profile.Output.Directory)
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
			displayHandle.tui.SetTransportStatus(display.StatusFailed)
			jackRunning = false
			reaper.Reap()
		} else {
			reaper.Callback("stop jack server", audioServer.StopServer)
		}
	} else {
		// TODO: make sure jackd is running prior to startup
	}

	return jackRunning
}
