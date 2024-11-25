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
package audio

import (
	"errors"
	"log/slog"
	"os"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

type OutputFile struct {
	ChannelName  string
	FilePath     string
	FileName     string
	Enabled      bool
	InputPorts   []*Port
	FileHandle   *os.File
	Encoder      *wav.Encoder
	ChannelCount int
	BitDepth     int
	SampleRate   int
	FileOpen     bool
}

func (of *OutputFile) GetWriteBuffers() []chan float32 {
	buffers := make([]chan float32, len(of.InputPorts))

	for i, port := range of.InputPorts {
		if port != nil && port.buffer != nil {
			buffers[i] = port.buffer
		}
	}

	return buffers
}

func (of *OutputFile) Close() {
	if of.FileOpen {
		slog.Info("Closing file " + of.FileName)
		if of.Encoder != nil {
			of.Encoder.Close()
		}

		if of.FileHandle != nil {
			of.FileHandle.Close()
		}

		of.FileOpen = false
	}
}

func (of *OutputFile) Write(buf *audio.IntBuffer) error {
	if !of.FileOpen {
		return errors.New("output file is already closed, " + of.FileName)
	}

	return of.Encoder.Write(buf)
}
