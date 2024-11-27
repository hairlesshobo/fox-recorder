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
package display

import (
	"fox-audio/model"
	"log/slog"
)

type UI interface {
	Initalize()
	Start()
	Shutdown()
	IsShutdown() bool
	WaitForShutdown()
	SetTransportStatus(status Status)
	SetDuration(duration float64)
	SetAudioFormat(format string)
	SetProfileName(value string)
	SetTakeName(value string)
	SetDirectory(value string)
	SetSessionSize(size uint64)
	IncrementErrorCount()
	UpdateSignalLevels(levels []model.SignalLevel)
	SetChannelArmStatus(channel int, armed bool)
	SetOutputFiles(outputFiles []model.UiOutputFile)
	UpdateOutputFileSizes(sizes []uint64)
	SetChannelCount(channelCount int)
	WriteLevelLog(level slog.Level, message string)
	SetAudioLoad(percent int)
	SetDiskUsage(percent int)
	SetBufferUtilization(percent int)
	SetDiskLoad(percent int)
	SetCycleBuffer(percent int)
}
