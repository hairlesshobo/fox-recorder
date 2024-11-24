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
package model

type Profile struct {
	AudioServer ProfileAudioServer `yaml:"audio_server"`
	Output      ProfileOutput      `yaml:"output"`
	Channels    []ProfileChannel   `yaml:"channels"`
}

type ProfileAudioServer struct {
	Interface         []string `yaml:"interface"`
	SampleRate        int      `yaml:"sample_rate"`
	FramesPerPeriod   int      `yaml:"frames_per_period"`
	BufferSizeSeconds float64  `yaml:"buffer_size_seconds"`
	MinimumWriteSize  float64  `yaml:"minimum_write_size"`
}

type ProfileChannel struct {
	Ports       []int  `yaml:"ports"`
	ChannelName string `yaml:"channel_name"`
	Enabled     bool   `yaml:"enabled"`
}

type ProfileOutput struct {
	DirectoryTemplate string `yaml:"directory_template"`
	Format            string `yaml:"format"`
	BitDepth          int    `yaml:"bit_depth"`

	// these are calculated at runtime and used internally, but
	// not able to be set in the profile
	Directory string
	Take      string
}
