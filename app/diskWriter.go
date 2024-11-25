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

	"fox-audio/model"
	"fox-audio/reaper"
	"fox-audio/util"

	"github.com/go-audio/audio"
)

func startDiskWriter(profile *model.Profile) {
	reaper.Register("disk writer")

	go diskWriter(profile)
}

// func getSamplesFromBuffer(sampleCount int, writeBuffer chan float32) []float32 {
// 	// TODO: Make sure this isn't necessary (trying to optimize each write cycle)
// 	currentBufferLen := len(writeBuffer)

// 	if currentBufferLen < sampleCount {
// 		// this should never happen
// 		slog.Error(fmt.Sprintf("Requested %d samples but only have %d", sampleCount, currentBufferLen))
// 	}

// 	samples := make([]float32, sampleCount)

// 	for i := 0; i < sampleCount; i++ {
// 		// populate the sample
// 		sample := <-writeBuffer
// 		samples[i] = sample
// 	}

// 	return samples
// }

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
				writeCycle(profile, true)
				break out
			}

			time.Sleep(1 * time.Millisecond)
		}
	}

	reaper.Done("disk writer")
}

func writeCycle(profile *model.Profile, finish bool) bool {
	requiredSamples := int(profile.Output.MinimumWriteSize * float64(profile.AudioServer.SampleRate))
	samplesToRead := requiredSamples
	factor := float32(math.Pow(2, float64(profile.Output.BitDepth)-1)) - 1.0

	// check if enough to write
	if finish ||
		(len(outputFiles) > 0 && len(outputFiles[0].GetWriteBuffers()[0]) > requiredSamples) {

		// disk load statistics
		if stats.diskProcessLastEndTime > 0 {
			if len(stats.diskProcessIdleChan) < cap(stats.diskProcessIdleChan) {
				stats.diskProcessIdleChan <- time.Now().UnixMicro() - stats.diskProcessLastEndTime
			}
		}
		stats.diskProcessLastStartTime = time.Now().UnixMicro()

		for _, channel := range outputFiles {
			// TODO: make sure channel is enabled
			writeBuffers := channel.GetWriteBuffers()

			if len(writeBuffers[0]) > requiredSamples || finish {
				if finish {
					samplesToRead = len(writeBuffers[0])
					slog.Debug(fmt.Sprintf("disk writer: reading remaining buffer samples: %d", samplesToRead))
				}

				util.TraceLog(fmt.Sprintf("Writing %d samples to file", samplesToRead))

				if !channel.FileOpen {
					slog.Error("Cannot write to closed file, " + channel.FileName)
					reaper.Reap()
					return false
				}

				for _, writeBuffer := range writeBuffers {
					// TODO: do i want to limit this to the requiredSampels count or just drain what it has left?

					//
					// previous version
					//

					// fBuf := &audio.Float32Buffer{
					// 	Data: getSamplesFromBuffer(samplesToRead, writeBuffer),
					// 	Format: &audio.Format{
					// 		NumChannels: int(channel.ChannelCount),
					// 		SampleRate:  int(channel.SampleRate),
					// 	},
					// }

					// transforms.PCMScaleF32(fBuf, profile.Output.BitDepth)

					// iBuf := fBuf.AsIntBuffer()

					// // TODO: add interleave support
					// channel.Write(iBuf)

					//
					// new version
					//
					buf := &audio.IntBuffer{
						Data: make([]int, samplesToRead),
						Format: &audio.Format{
							NumChannels: int(channel.ChannelCount),
							SampleRate:  int(channel.SampleRate),
						},
					}

					// TODO: does this actually support 24 bit??
					// for each sample we load, scale and cast to int
					for i := 0; i < samplesToRead; i++ {
						// populate the sample
						// buf.Data[i] = <-writeBuffer
						buf.Data[i] = int(<-writeBuffer * factor)
					}

					// TODO: add interleave support
					channel.Write(buf)
				}

				if finish {
					channel.Close()
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
