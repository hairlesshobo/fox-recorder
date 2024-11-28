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

type OutputType int

const (
	OutputTUI OutputType = iota
	OutputJSON
	// OutputText
)

type CommandLineArgs struct {
	Simulate             bool
	SimulateChannelCount int
	SimulateFreezeMeters bool

	ProfileName string
	ConfigFile  string
	OutputType  string
}

type Config struct {
	JackdBinary                  string     `yaml:"jackd_binary,omitempty"`
	VerboseJackServer            bool       `yaml:"verbose_jack_server,omitempty"`
	JackClientName               string     `yaml:"jack_client_name,omitempty"`
	ProfileDirectory             string     `yaml:"profile_directory,omitempty"`
	LogLevel                     int        `yaml:"log_level,omitempty"`
	HardwarePortConnectionPrefix string     `yaml:"hardware_port_connection_prefix,omitempty"`
	OutputType                   OutputType `yaml:"output_Type,omitempty"`

	SimulationOptions *SimulationOptions `yaml:"simulation_options"`
}

type SimulationOptions struct {
	EnableSimulation bool `yaml:"enable,omitempty"`
	FreezeMeters     bool `yaml:"freeze_meters,omitempty"`
	ChannelCount     int  `yaml:"channel_count,omitempty"`
}
