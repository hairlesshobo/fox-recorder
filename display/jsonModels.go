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

type JsonStatus struct {
	MessageType string `json:"message_type"`

	Status string `json:"status"`

	Duration    float64 `json:"duration"`
	Format      string  `json:"format"`
	SessionSize uint64  `json:"session_size"`
	ErrorCount  int     `json:"error_count"`
	ProfileName string  `json:"profile_name"`
	TakeName    string  `json:"take_name"`
	Directory   string  `json:"directory"`

	DiskUsedPct        int `json:"disk_used_pct"`
	BufferUsedPct      int `json:"buffer_used_pct"`
	CycleBufferUsedPct int `json:"cycle_buffer_used_pct"`
	AudioLoadPct       int `json:"audio_load_pct"`
	DiskLoadPct        int `json:"disk_load_pct"`
}

type JsonLog struct {
	MessageType string `json:"message_type"`

	Date    string `json:"date"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

type JsonLevels struct {
	MessageType string `json:"message_type"`

	Ports []JsonLevelPort `json:"ports"`
}

type JsonLevelPort struct {
	Name  string `json:"name"`
	Level int    `json:"level"`
}

type JsonOutputFiles struct {
	MessageType string `json:"message_type"`

	Files []JsonOutputFile `json:"files"`
}

type JsonOutputFile struct {
	Name  string   `json:"name"`
	Ports []string `json:"ports"`
	Size  uint64   `json:"size"`
}
