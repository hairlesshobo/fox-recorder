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
	"fox-audio/util"
	"math"
	"os"
)

var (
	stats statistics
)

type statistics struct {
	lastStartTime      int64
	lastEndTime        int64
	processElapsedChan chan int64
	processIdleChan    chan int64
}

func initStatistics() {
	stats = statistics{
		processElapsedChan: make(chan int64, 30),
		processIdleChan:    make(chan int64, 30),
	}

	// audio engine load calculations
	util.ProcessOnInterval(250, func() {
		idleTimeAvg := util.GetChanAverage(stats.processIdleChan)
		processTimeAvg := util.GetChanAverage(stats.processElapsedChan)
		avgAudioLoad := processTimeAvg / idleTimeAvg

		displayHandle.tui.SetAudioLoad(int(avgAudioLoad * 100.0))
		util.TraceLog(fmt.Sprintf("Idle time: %0.0f us, Process time: %0.0f us, load %0.3f%%", idleTimeAvg, processTimeAvg, avgAudioLoad))
	})

	// disk space utilization
	util.ProcessOnInterval(1000, func() {
		// TODO: this should point to recording directory
		wd, _ := os.Getwd()

		diskInfo := util.GetDiskSpace(wd)
		displayHandle.tui.SetDiskUsage(int(math.Round(diskInfo.UsedPct * 100.0)))

		util.TraceLog(fmt.Sprintf("Disk total: %d B, Disk Used: %d B, Disk free: %d B, used %0.2f%%", diskInfo.Size, diskInfo.Used, diskInfo.Free, diskInfo.UsedPct))
	})
}
