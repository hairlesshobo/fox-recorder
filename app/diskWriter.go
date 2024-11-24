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
	reaper.Register("disk writer")

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
			if !writeCycle(profile, false) {
				break out
			}

		default:
			// waiting for data, check for reapage and sleep briefly
			if reaper.Reaped() {
				slog.Debug("diskwriter: reap caught, finish writing buffer")
				// TODO: we were reaped.. clean up
				writeCycle(profile, true)
				break out
			}

			time.Sleep(10 * time.Millisecond)
		}
	}

	reaper.Done("disk writer")
}

func writeCycle(profile *model.Profile, finish bool) bool {
	for _, channel := range outputFiles {
		requiredSamples := int(profile.AudioServer.MinimumWriteSize * float64(profile.AudioServer.SampleRate))

		writeBuffers := channel.GetWriteBuffers()

		// check if enough to write, then write
		if len(writeBuffers[0]) > requiredSamples || finish {
			samplesToRead := requiredSamples

			if finish {
				samplesToRead = len(writeBuffers[0])
				slog.Debug("disk writer: reading remaining buffer samples: " + string(samplesToRead))
			}

			slog.Debug(fmt.Sprintf("Writing %d samples to file", samplesToRead))

			if !channel.FileOpen {
				slog.Error("Cannot write to closed file, " + channel.FileName)
				reaper.Reap()
				return false
			}

			for _, writeBuffer := range writeBuffers {
				// TODO: do i want to limit this to the requiredSampels count or just drain what it has left?
				fBuf := &audio.Float32Buffer{
					Data: getSamplesFromBuffer(samplesToRead, writeBuffer),
					Format: &audio.Format{
						NumChannels: int(channel.ChannelCount),
						SampleRate:  int(channel.SampleRate),
					},
				}

				transforms.PCMScaleF32(fBuf, profile.Output.BitDepth)

				iBuf := fBuf.AsIntBuffer()

				// TODO: add interleave support
				channel.Write(iBuf)
			}

			if finish {
				channel.Close()
			}
		}
	}

	return true
}
