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
	"os"
	"strconv"
	"time"

	"fox-audio/audio"
	"fox-audio/display"
	"fox-audio/model"
	"fox-audio/reaper"
	"fox-audio/shared"
)

// TODO: move to config
const jackClientName = "fox"

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

func runEngine(profile *model.Profile, simulate bool, simulateFreezeMeters bool, simulateChannelCount int) {
	displayHandle.tui = display.NewTui()
	displayHandle.tui.Initalize()
	displayHandle.tui.SetTransportStatus(3)
	displayHandle.tui.Start()
	reaper.Callback("tui", displayHandle.tui.Shutdown)

	statsShutdownChan := initStatistics(profile)
	reaper.Callback("stats", func() { statsShutdownChan <- true })

	reaper.Callback("wait", func() { time.Sleep(3 * time.Second) })

	// ConfigureTextLogger()
	ConfigureTuiLogger()
	// ConfigureFileLogger()

	if !simulate {
		audioServer = audio.NewServer(jackClientName, profile)
		audioServer.SetErrorCallback(jackError)
		audioServer.SetInfoCallback(jackInfo)

		shared.CatchSigint(func() {
			slog.Info("Caught sigint, calling reaper")
			reaper.Reap()
		})

		audioServer.StartServer()
		reaper.Callback("stop jack server", audioServer.StopServer)

		ports = audioServer.GetInputPorts()
		displayHandle.tui.SetChannelCount(len(ports))
		signalLevels = make([]*model.SignalLevel, len(ports))
		displayHandle.tui.SetTransportStatus(0)

		audioServer.Connect()
		reaper.Callback("disconnect jack server", audioServer.Disconnect)

		// only register input ports, for now
		audioServer.RegisterPorts(true, false)

		// set callbacks
		audioServer.SetProcessCallback(jackProcess)
		audioServer.SetXrunCallback(jackXrun)
		audioServer.SetShutdownCallback(func() { jackShutdown(audioServer) })

		audioServer.ActivateClient()

		audioServer.PrepareOutputFiles()
		outputFiles = audioServer.GetOutputFiles()
		startDiskWriter(profile)

		audioServer.ConnectPorts(true, false)

		sampleRateStr := strconv.FormatFloat(float64(audioServer.GetSampleRate())/1000.0, 'f', -1, 64)
		displayHandle.tui.SetAudioFormat(fmt.Sprintf("%dbit / %sKHz", profile.Output.BitDepth, sampleRateStr))
		displayHandle.tui.SetTransportStatus(display.StatusRecording)
	}

	// this blocks until the jack connection shuts down
	if simulate {
		displayHandle.tui.SetChannelCount(simulateChannelCount)
		startSimulation(simulateFreezeMeters, simulateChannelCount)
	}

	reaper.Callback("shutdown status", func() { displayHandle.tui.SetTransportStatus(display.StatusShuttingDown) })
	reaper.Wait()
}
