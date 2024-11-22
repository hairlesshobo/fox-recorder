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
package reaper

import (
	"slices"
	"sync"
)

var (
	reapRequested   chan bool
	reaperCallbacks []func()
	reaperWaitgroup sync.WaitGroup
)

func init() {
	reapRequested = make(chan bool, 1)
	reaperCallbacks = make([]func(), 0)
	reaperWaitgroup = sync.WaitGroup{}
}

func Reaped() bool {
	return len(reapRequested) > 0
}

func Reap() {
	if len(reapRequested) == 0 {
		reapRequested <- true

		callbacksReversed := slices.Clone(reaperCallbacks)
		slices.Reverse(callbacksReversed)

		for _, callback := range callbacksReversed {
			callback()
		}
	}
}

func Callback(callback func()) {
	reaperCallbacks = append(reaperCallbacks, callback)
}

func Register() {
	reaperWaitgroup.Add(1)
}

func Done() {
	reaperWaitgroup.Done()
}

func Wait() {
	reaperWaitgroup.Wait()
}
