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
	ports         []*audio.Port
	audioServer   *audio.JackServer
)

func runEngine(profile *model.Profile, simulate bool, simulateFreezeMeters bool, simulateChannelCount int) {
	displayHandle.tui = display.NewTui()
	displayHandle.tui.Initalize()
	displayHandle.tui.SetTransportStatus(3)
	displayHandle.tui.Start()

	initStatistics()

	// f, err := os.Create("/Users/flip/projects/personal/fox-recorder/fox.log")

	// if err != nil {
	// 	panic(err)
	// }

	// handler := slog.NewTextHandler(f, &slog.HandlerOptions{
	// 	Level: slog.LevelDebug,
	// })
	// logger := slog.New(handler)
	// slog.SetDefault(logger)

	handler := shared.NewTuiLogHandler(displayHandle.tui, slog.LevelDebug)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	shared.HijackLogging()
	shared.EnableSlogLogging()

	if !simulate {
		audioServer = audio.NewServer(jackClientName, profile)
		audioServer.SetErrorCallback(jackError)
		audioServer.SetInfoCallback(jackInfo)

		shared.CatchSigint(func() {
			slog.Info("Caught sigint, calling reaper")
			reaper.Reap()
		})

		audioServer.StartServer()
		reaper.Callback(audioServer.StopServer)

		ports = audioServer.GetInputPorts()
		displayHandle.tui.SetChannelCount(len(ports))
		signalLevels = make([]*model.SignalLevel, len(ports))
		displayHandle.tui.SetTransportStatus(0)

		audioServer.Connect()
		reaper.Callback(audioServer.Disconnect)

		// only register input ports, for now
		audioServer.RegisterPorts(true, false)

		// set callbacks
		audioServer.SetProcessCallback(jackProcess)
		audioServer.SetXrunCallback(jackXrun)
		audioServer.SetShutdownCallback(func() { jackShutdown(audioServer) })

		audioServer.ActivateClient()

		audioServer.ConnectPorts(true, false)

		audioServer.PrepareOutputFiles()
		// TODO: wire up output files
		// GetOutputFiles

		displayHandle.tui.SetAudioFormat(fmt.Sprintf("%0.1fKHz", float64(audioServer.GetSampleRate())/1000.0))

		slog.Info("Input ports connected")
	}

	// TODO: connect port(s)

	// this blocks until the jack connection shuts down
	if simulate {
		displayHandle.tui.SetChannelCount(simulateChannelCount)
		startSimulation(simulateFreezeMeters, simulateChannelCount)
	}
	// displayHandle.tui.WaitForShutdown()
	// audioServer.StopServer()
	reaper.Wait()
}
