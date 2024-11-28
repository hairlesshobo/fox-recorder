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

	"fox-audio/model"
	"fox-audio/reaper"
	"fox-audio/util"
)

var (
	signalLevels     []model.SignalLevel
	cycleDoneChannel chan bool

	transportRecord bool
)

func init() {
	transportRecord = false
}

func jackError(message string) {
	if reaper.Reaped() {
		slog.Warn("JACK client: " + message)
	} else {
		slog.Error("JACK client: " + message)
	}
}

func jackInfo(message string) {
	slog.Info("JACK client: " + message)
}

func jackShutdown() {
	slog.Info("JACK client: connection is shutting down")
	transportRecord = false
	reaper.Reap()
}

func jackXrun() int {
	slog.Error("JACK client: xrun occurred")

	return 0
}

func jackProcess(nframes uint32) int {
	// audio load statistics
	if stats.jackProcessLastEndTime > 0 {
		if len(stats.jackProcessIdleChan) < cap(stats.jackProcessIdleChan) {
			stats.jackProcessIdleChan <- time.Now().UnixMicro() - stats.jackProcessLastEndTime
		}
	}

	stats.jackProcessLastStartTime = time.Now().UnixMicro()

	if !reaper.Reaped() && transportRecord {
		stats.framesProcessed += uint64(nframes)
	}

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
			signalLevels[portNum] = model.SignalLevel{
				Instant: int(util.AmplitudeToDb(sigLevel)),
			}

			// TODO: make a transport class
			if !transportRecord {
				continue
			}

			if port.IsArmed() {
				writeBuffer := port.GetWriteBuffer()
				if cap(writeBuffer) > 0 {
					if (len(writeBuffer) + int(nframes)) < cap(writeBuffer) {
						// stats.samplesProcessed += uint64(nframes)

						for _, sample := range samplesIn {
							writeBuffer <- float32(sample)
						}
					} else {
						slog.Error("No space left in write buffer!!")
					}
				}
			}
		} else {
			signalLevels[portNum] = model.SignalLevel{
				Instant: -150,
			}
		}
	}

	displayHandle.UpdateSignalLevels(signalLevels)

	if !reaper.Reaped() {
		cycleDoneChannel <- true
	}

	// audio load statistics
	stats.jackProcessLastEndTime = time.Now().UnixMicro()
	if len(stats.jackProcessElapsedChan) < cap(stats.jackProcessElapsedChan) {
		stats.jackProcessElapsedChan <- stats.jackProcessLastEndTime - stats.jackProcessLastStartTime

	}

	return 0
}
