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
	"github.com/hairlesshobo/go-jack"
)

type PortDirection int8

const (
	In PortDirection = iota
	Out
)

type Port struct {
	portDirection PortDirection
	myName        string
	connected     bool
	jackName      string
	jackPort      *jack.Port
	buffer        chan float32
}

func newPort(direction PortDirection, myName string, jackName string) *Port {
	return &Port{
		portDirection: direction,
		myName:        myName,
		jackName:      jackName,
		connected:     false,
	}
}

func (port *Port) setJackPort(jackPort *jack.Port) {
	port.jackPort = jackPort
}

func (port *Port) GetJackPort() *jack.Port {
	return port.jackPort
}

func (port *Port) GetJackBuffer(nframes uint32) []jack.AudioSample {
	return port.jackPort.GetBuffer(nframes)
}

func (port *Port) AllocateBuffer(size int) bool {
	if cap(port.buffer) > 0 {
		return false
	}

	port.buffer = make(chan float32, size)

	return true
}

func (port *Port) GetWriteBuffer() chan float32 {
	return port.buffer
}
