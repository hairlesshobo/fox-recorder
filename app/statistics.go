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

var (
	stats statistics
)

type statistics struct {
	lastStartTime      int64
	lastEndTime        int64
	processElapsedChan chan int64
	processIdleChan    chan int64
	shutdownChan       chan bool
	samplesProcessed   uint64
}

func initStatistics(profile *model.Profile) chan bool {
	stats = statistics{
		processElapsedChan: make(chan int64, 30),
		processIdleChan:    make(chan int64, 30),
		shutdownChan:       make(chan bool, 5),
		samplesProcessed:   0,
	}

	channels := 0

	for _, channel := range profile.Channels {
		channels += len(channel.Ports)
	}

	// audio engine load calculations
	processOnInterval("audio engine load stats", stats.shutdownChan, 250, func() {
		idleTimeAvg := util.GetChanAverage(stats.processIdleChan)
		processTimeAvg := util.GetChanAverage(stats.processElapsedChan)
		avgAudioLoad := processTimeAvg / idleTimeAvg

		if !math.IsNaN(avgAudioLoad) {
			displayHandle.tui.SetAudioLoad(int(avgAudioLoad * 100.0))
			util.TraceLog(fmt.Sprintf("Idle time: %0.0f us, Process time: %0.0f us, load %0.3f%%", idleTimeAvg, processTimeAvg, avgAudioLoad))

			// TODO: calculate disk write load (time writing / time idle)
		}
	})

	// disk space utilization & session size
	processOnInterval("disk space & session size stats", stats.shutdownChan, 1000, func() {
		channelCount := 0
		for _, channel := range profile.Channels {
			channelCount += len(channel.Ports)
		}

		fileCount := len(profile.Channels)

		// get bytes read from jack
		usedBytesRaw := (stats.samplesProcessed * uint64(profile.Output.BitDepth)) / 8
		usedBytesRaw *= uint64(channelCount)

		// add 44 for each wav file header
		usedBytes := usedBytesRaw + (uint64(fileCount) * 44)
		displayHandle.tui.SetSessionSize(usedBytes)

		diskInfo := util.GetDiskSpace(profile.Output.Directory)
		displayHandle.tui.SetDiskUsage(int(math.Round(diskInfo.UsedPct * 100.0)))

		util.TraceLog(fmt.Sprintf("Disk total: %d B, Disk Used: %d B, Disk free: %d B, used %0.2f%%", diskInfo.Size, diskInfo.Used, diskInfo.Free, diskInfo.UsedPct))
	})

	// buffer utilization
	processOnInterval("buffer stats", stats.shutdownChan, 200, func() {
		sum := float64(0.0)
		count := 0

		for _, port := range ports {
			buffer := port.GetWriteBuffer()

			sum += float64(len(buffer)) / float64(cap(buffer))
			count += 1
		}

		bufferAvg := sum / float64(count)

		if !math.IsNaN(bufferAvg) {
			displayHandle.tui.SetBufferUtilization(int(math.Round(bufferAvg * 100.0)))
			util.TraceLog(fmt.Sprintf("buffer: %0.2f%%", bufferAvg))
		}
	})

	processOnInterval("recording duration stats", stats.shutdownChan, 50, func() {
		duration := float64(stats.samplesProcessed) / float64(profile.AudioServer.SampleRate)
		displayHandle.tui.SetDuration(duration)
	})

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
