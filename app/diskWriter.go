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
	"time"

	"fox-audio/model"
	"fox-audio/reaper"

	"github.com/go-audio/audio"
	"github.com/go-audio/transforms"
)

func startDiskWriter(profile *model.Profile) {
	reaper.Register()
	go diskWriter(profile)
}

func getSamplesFromBuffer(sampleCount int, writeBuffer chan float32) []float32 {
	currentBufferLen := len(writeBuffer)

	if currentBufferLen < sampleCount {
		// TODO: track this error
		// this should never happen
		slog.Warn(fmt.Sprintf("Requested %d samples but only have %d", sampleCount, currentBufferLen))
	}

	samples := make([]float32, sampleCount)

	for i := 0; i < sampleCount; i++ {
		sample := <-writeBuffer
		samples[i] = sample
	}

	return samples
}

func diskWriter(profile *model.Profile) {
out:
	for {
		select {
		case <-cycleDoneChannel:
			// check if enough to write, then write
			for _, channel := range outputFiles {
				requiredSamples := int(profile.AudioServer.MinimumWriteSize * float64(profile.AudioServer.SampleRate))

				writeBuffers := channel.GetWriteBuffers()

				if len(writeBuffers[0]) > requiredSamples {
					slog.Debug("Writing samples to file")

					// TODO: Need to make sure the file isn't closed

					for _, writeBuffer := range writeBuffers {
						// TODO: do i want to limit this to the requiredSampels count or just drain what it has left?
						wavFormat := &audio.Format{
							NumChannels: int(channel.ChannelCount),
							SampleRate:  int(channel.SampleRate),
						}

						samples := getSamplesFromBuffer(requiredSamples, writeBuffer)

						fBuf := &audio.Float32Buffer{
							Data:   samples,
							Format: wavFormat,
						}

						transforms.PCMScaleF32(fBuf, profile.Output.BitDepth)

						iBuf := fBuf.AsIntBuffer()

						// TODO: add interleave support
						channel.Write(iBuf)
					}
				}
			}
		default:
			// waiting for data, check for reapage and sleep briefly
			if reaper.Reaped() {
				// TODO: we were reaped.. clean up
				break out
			}

			time.Sleep(10 * time.Millisecond)
		}
	}

	reaper.Done()
}
