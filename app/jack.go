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
	"log/slog"
	"time"

	"fox-audio/audio"
	"fox-audio/model"
	"fox-audio/reaper"
	"fox-audio/util"
)

var (
	signalLevels     []*model.SignalLevel
	cycleDoneChannel chan bool
)

func init() {
	// TODO: does this need to change - do we need to track this channel fill ratio too?
	cycleDoneChannel = make(chan bool, 30)
}

func jackError(message string) {
	slog.Error("JACK client: " + message)
}

func jackInfo(message string) {
	slog.Info("JACK client: " + message)
}

func jackShutdown(server *audio.JackServer) {
	slog.Info("JACK client: connection is shutting down")
	server.StopServer()
}

func jackXrun() int {
	slog.Error("JACK client: xrun occurred")

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
	for portNum, port := range ports {

		// get the incoming audio samples
		samplesIn := port.GetJackBuffer(nframes)

		sigLevel := float32(-1.0)

		for frame := range nframes {
			sample := float32(samplesIn[frame])

			if sample > sigLevel {
				sigLevel = sample
			}
		}

		if !reaper.Reaped() {
			stats.samplesProcessed += uint64(nframes)

			signalLevels[portNum] = &model.SignalLevel{
				Instant: int(util.AmplitudeToDb(sigLevel)),
			}

			// TODO: add a check to make sure recording is enabled

			// if it has a buffer, its enabled
			// TODO: change this to an explicit Enabled flag?
			writeBuffer := port.GetWriteBuffer()
			if cap(writeBuffer) > 0 {
				if (len(writeBuffer) + int(nframes)) < cap(writeBuffer) {
					for _, sample := range samplesIn {
						writeBuffer <- float32(sample)
					}

				} else {
					slog.Error("No space left in write buffer!!")
				}
			}
		} else {
			signalLevels[portNum] = &model.SignalLevel{
				Instant: -150,
			}
		}
	}

	displayHandle.tui.UpdateSignalLevels(signalLevels)

	cycleDoneChannel <- true

	// audio load statistics
	stats.lastEndTime = time.Now().UnixMicro()
	if len(stats.processElapsedChan) < cap(stats.processElapsedChan) {
		stats.processElapsedChan <- stats.lastEndTime - stats.lastStartTime

	}
	return 0
}
