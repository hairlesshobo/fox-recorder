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
	"fox-audio/audio"
	"fox-audio/model"
	"fox-audio/util"
	"log/slog"
	"time"
)

func jackError(message string) {
	slog.Error("JACK: " + message)
}

func jackInfo(message string) {
	slog.Info("JACK: " + message)
}

func jackShutdown(server *audio.JackServer) {
	slog.Info("JACK connection shutting down")
	server.StopServer()
}

func jackXrun() int {
	slog.Error("xrun")

	return 0
}

func jackProcess(nframes uint32) int {
	// audio load statistics
	if stats.lastEndTime > 0 {
		if len(stats.processIdleChan) < cap(stats.processIdleChan) {
			stats.processIdleChan <- time.Now().UnixMicro() - stats.lastEndTime
		}
	}

	stats.lastStartTime = time.Now().UnixMicro()

	// loop through the input channels
	levels := make([]*model.SignalLevel, len(ports))

	for portNum, port := range ports {

		// get the incoming audio samples
		samplesIn := port.GetJackPort().GetBuffer(nframes)

		sigLevel := -150.0

		for frame := range nframes {
			sample := (float64)(samplesIn[frame])
			frameLevel := util.AmplitudeToDb(sample)

			if frameLevel > sigLevel {
				sigLevel = frameLevel
			}
		}

		levels[portNum] = &model.SignalLevel{
			Instant: int(sigLevel),
		}
	}

	displayHandle.tui.UpdateSignalLevels(levels)

	// audio load statistics
	stats.lastEndTime = time.Now().UnixMicro()
	if len(stats.processElapsedChan) < cap(stats.processElapsedChan) {
		stats.processElapsedChan <- stats.lastEndTime - stats.lastStartTime

	}
	return 0
}
