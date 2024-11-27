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
	"math"
	"time"

	"fox-audio/model"
	"fox-audio/reaper"
	"fox-audio/util"
)

const (
	samplesToAverage = 5
)

var (
	stats statistics
)

type statistics struct {
	jackProcessLastStartTime int64
	jackProcessLastEndTime   int64
	jackProcessElapsedChan   chan int64
	jackProcessIdleChan      chan int64

	diskProcessLastStartTime int64
	diskProcessLastEndTime   int64
	diskProcessElapsedChan   chan int64
	diskProcessIdleChan      chan int64

	shutdownChan    chan bool
	framesProcessed uint64
	// samplesProcessed uint64

	diskPerformance   []float64
	bufferUtilization []float64
	audioLoad         []float64
	cycleLoad         []float64
}

func initStatistics(profile *model.Profile) chan bool {
	stats = statistics{
		jackProcessElapsedChan: make(chan int64, 1),
		jackProcessIdleChan:    make(chan int64, 1),

		diskProcessElapsedChan: make(chan int64, 1),
		diskProcessIdleChan:    make(chan int64, 1),

		shutdownChan: make(chan bool, 5),
		// samplesProcessed: 0,

		diskPerformance:   make([]float64, samplesToAverage),
		bufferUtilization: make([]float64, samplesToAverage),
		audioLoad:         make([]float64, samplesToAverage),
		cycleLoad:         make([]float64, samplesToAverage),
	}

	channels := 0

	for _, channel := range profile.Channels {
		channels += len(channel.Ports)
	}

	// session size
	processOnInterval("session size stats", stats.shutdownChan, 500, func() {
		usedBytes := uint64(0)

		outputFileSizes := make([]uint64, len(outputFiles))
		for i, outputFile := range outputFiles {
			outputFileSizes[i] = uint64(outputFile.Encoder.WrittenBytes)
			usedBytes += uint64(outputFile.Encoder.WrittenBytes)
		}

		displayHandle.tui.UpdateOutputFileSizes(outputFileSizes)
		displayHandle.tui.SetSessionSize(usedBytes)

		// get bytes read from jack
		// usedBytesRaw := (stats.samplesProcessed * uint64(profile.Output.BitDepth)) / 8

		// add 44 for each wav file header
		// usedBytes := usedBytesRaw + (uint64(len(profile.Channels)) * 44)
		// displayHandle.tui.SetSessionSize(usedBytes)
	})

	// disk space utilization
	processOnInterval("disk space", stats.shutdownChan, 5000, func() {
		diskInfo := util.GetDiskSpace(profile.Output.Directory)
		displayHandle.tui.SetDiskUsage(int(math.Round(diskInfo.UsedPct * 100.0)))

		util.TraceLog(fmt.Sprintf("Disk total: %d B, Disk Used: %d B, Disk free: %d B, used %0.2f%%", diskInfo.Size, diskInfo.Used, diskInfo.Free, diskInfo.UsedPct*100.0))
	})

	processOnInterval("combined stats", stats.shutdownChan, 100, func() {
		// buffer utilization
		bufferSum := float64(0.0)
		bufferCount := 0

		for _, port := range ports {
			if port.IsArmed() {
				buffer := port.GetWriteBuffer()

				bufferSum += float64(len(buffer)) / float64(cap(buffer))
				bufferCount += 1
			}
		}

		bufferPct := bufferSum / float64(bufferCount)

		if !math.IsNaN(bufferPct) {
			stats.bufferUtilization = pushStatistic(stats.bufferUtilization, bufferPct, samplesToAverage)
			avgBufferPct := math.Round(averageStatistic(stats.cycleLoad) * 100.0)

			displayHandle.tui.SetBufferUtilization(int(avgBufferPct))
			util.TraceLog(fmt.Sprintf("buffer: %0.2f%%", avgBufferPct))
		}
	})

	// disk load
	go func() {
		for {
			idleDuration := <-stats.diskProcessIdleChan
			writeDuration := <-stats.diskProcessElapsedChan

			// calculate disk load
			diskLoadPct := float64(writeDuration) / (float64(idleDuration) + float64(writeDuration))

			if !math.IsNaN(diskLoadPct) {
				stats.diskPerformance = pushStatistic(stats.diskPerformance, diskLoadPct, samplesToAverage)
				avgDiskLoadPct := math.Round(averageStatistic(stats.cycleLoad) * 100.0)

				displayHandle.tui.SetDiskLoad(int(avgDiskLoadPct))
				util.TraceLog(fmt.Sprintf("disk Idle time: %d us, Process time: %d us, load %0.3f%%", idleDuration, writeDuration, avgDiskLoadPct))
			}
		}
	}()

	// audio engine load
	// this triggers on every call of "process" so it needs to be fast since it
	// runs in sync with that method
	go func() {
		for {
			idleDuration := <-stats.jackProcessIdleChan
			writeDuration := <-stats.jackProcessElapsedChan

			// calculate disk load
			audioLoadPct := float64(writeDuration) / (float64(idleDuration) + float64(writeDuration))

			if !math.IsNaN(audioLoadPct) {
				stats.audioLoad = pushStatistic(stats.audioLoad, audioLoadPct, samplesToAverage)
				avgAudioLoadPct := math.Round(averageStatistic(stats.audioLoad) * 100.0)

				displayHandle.tui.SetAudioLoad(int(avgAudioLoadPct))
				util.TraceLog(fmt.Sprintf("audio Idle time: %d us, Process time: %d us, load %0.3f%%", idleDuration, writeDuration, avgAudioLoadPct))
			}

			// cycle buffer
			cycleBuffer := float64(len(cycleDoneChannel)) / float64(cap(cycleDoneChannel))

			if !math.IsNaN(cycleBuffer) {
				stats.cycleLoad = pushStatistic(stats.cycleLoad, cycleBuffer, samplesToAverage)
				avgCycleBuffer := math.Round(averageStatistic(stats.cycleLoad) * 100.0)

				displayHandle.tui.SetCycleBuffer(int(avgCycleBuffer))
				util.TraceLog(fmt.Sprintf("cycle buffer: %03f%%", cycleBuffer))
			}

			// recording duration
			duration := float64(stats.framesProcessed) / float64(profile.AudioServer.SampleRate)
			displayHandle.tui.SetDuration(duration)
		}
	}()

	return stats.shutdownChan
}

func processOnInterval(name string, shutdownChan chan bool, milliseconds int, process func()) {
	reaper.Register(name)

	go func() {
		process()

		t := time.NewTicker(time.Duration(milliseconds) * time.Millisecond)

		for range t.C {
			if len(shutdownChan) > 0 {
				break
			}

			process()
		}

		reaper.Done(name)
	}()
}

func pushStatistic(slice []float64, newValue float64, maxSamples int) []float64 {
	slice = append(slice, newValue)

	if len(slice) > maxSamples {
		slice = slice[1:]
	}

	return slice
}

func averageStatistic(slice []float64) float64 {
	sum := 0.0
	count := 0

	for i := 0; i < len(slice); i++ {
		sum += slice[i]
		count++
	}

	return sum / float64(count)
}
