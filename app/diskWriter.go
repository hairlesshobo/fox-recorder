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
	"time"

	"fox-audio/audio"
	"fox-audio/model"
	"fox-audio/reaper"
	"fox-audio/util"

	jackAudio "github.com/go-audio/audio"
)

func startDiskWriter(profile *model.Profile) {
	reaper.Register("disk writer")

	go diskWriter(profile)
}

func diskWriter(profile *model.Profile) {
	defer reaper.HandlePanic()

out:
	for {
		select {
		case <-cycleDoneChannel:
			if !writeCycle(profile, false) {
				break out
			}

		default:
			// waiting for data, check for reapage and sleep briefly
			if reaper.Reaped() {
				slog.Debug("diskwriter: reap caught, finish writing buffer")
				writeCycle(profile, true)
				break out
			}

			time.Sleep(1 * time.Millisecond)
		}
	}

	reaper.Done("disk writer")
}

func getFirstEnabledOutputFile() *audio.OutputFile {
	for _, file := range outputFiles {
		if file.Enabled {
			return file
		}
	}

	return nil
}

func writeCycle(profile *model.Profile, finish bool) bool {
	requiredSamples := int(profile.Output.MinimumWriteSize * float64(profile.AudioServer.SampleRate))
	samplesToRead := requiredSamples
	factor := float32(math.Pow(2, float64(profile.Output.BitDepth)-1)) - 1.0

	// check if enough to write
	if finish ||
		(len(outputFiles) > 0 && len(getFirstEnabledOutputFile().GetWriteBuffers()[0]) > requiredSamples) {

		// disk load statistics
		if stats.diskProcessLastEndTime > 0 {
			if len(stats.diskProcessIdleChan) < cap(stats.diskProcessIdleChan) {
				stats.diskProcessIdleChan <- time.Now().UnixMicro() - stats.diskProcessLastEndTime
			}
		}
		stats.diskProcessLastStartTime = time.Now().UnixMicro()

		for _, outputFile := range outputFiles {
			if !outputFile.Enabled {
				continue
			}

			writeBuffers := outputFile.GetWriteBuffers()

			if len(writeBuffers[0]) > requiredSamples || finish {
				if finish {
					samplesToRead = len(writeBuffers[0])
					util.TraceLog(fmt.Sprintf("disk writer: reading remaining buffer samples: %d", samplesToRead))
				}

				util.TraceLog(fmt.Sprintf("Writing %d samples to file", samplesToRead))

				if !outputFile.FileOpen {
					slog.Error("Cannot write to closed file, " + outputFile.FileName)
					reaper.Reap()
					return false
				}

				// allocate the encoder buffer
				buf := &jackAudio.IntBuffer{
					Data: make([]int, samplesToRead*outputFile.ChannelCount),
					Format: &jackAudio.Format{
						NumChannels: int(outputFile.ChannelCount),
						SampleRate:  int(outputFile.SampleRate),
					},
				}

				bufferIndex := 0

				// loop through the samples, then the channel buffers in order to interleave the output
				for sampleIndex := 0; sampleIndex < samplesToRead*outputFile.ChannelCount; sampleIndex += outputFile.ChannelCount {
					for bufferIndex = 0; bufferIndex < outputFile.ChannelCount; bufferIndex++ {
						// for each sample we load, scale and cast to int
						buf.Data[sampleIndex+bufferIndex] = int(<-writeBuffers[bufferIndex] * factor)
					}
				}

				outputFile.Write(buf)

				if finish {
					outputFile.Close()
				}
			}
		}

		// disk load statistics
		stats.diskProcessLastEndTime = time.Now().UnixMicro()
		if len(stats.diskProcessElapsedChan) < cap(stats.diskProcessElapsedChan) {
			stats.diskProcessElapsedChan <- stats.diskProcessLastEndTime - stats.diskProcessLastStartTime

		}
	}

	return true
}
