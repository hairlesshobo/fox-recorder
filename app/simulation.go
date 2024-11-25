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
	"math/rand/v2"
	"time"

	"fox-audio/model"
	"fox-audio/reaper"
)

func startSimulation(simulationOptions *model.SimulationOptions) {
	reaper.Register("simulation")

	go func() {
		t := time.NewTicker(150 * time.Millisecond)
		levels := make([]*model.SignalLevel, simulationOptions.ChannelCount)

		displayHandle.tui.SetTransportStatus(2)
		displayHandle.tui.SetAudioFormat("24 bit / 48k WAV")

		size := uint64(0)

		for range t.C {
			size += uint64(rand.IntN(5) * 1024 * 32)

			if reaper.Reaped() {
				break
			}

			for channel := range simulationOptions.ChannelCount {
				newLevel := (rand.IntN(70) + 0) * (-1)

				levels[channel] = &model.SignalLevel{
					Instant: int(newLevel),
				}
			}

			// Queue draw
			displayHandle.tui.UpdateSignalLevels(levels)
			displayHandle.tui.SetSessionSize(size)

			if simulationOptions.FreezeMeters {
				break
			}
		}

		reaper.Done("simulation")
	}()
}
