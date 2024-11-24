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
	"log/slog"
	"slices"
	"sync"
)

var (
	reapRequested       chan bool
	reaperCallbacks     []callback
	reaperRegistrations []string
	reaperWaitgroup     sync.WaitGroup
)

type callback struct {
	name         string
	callbackFunc func()
}

func init() {
	reapRequested = make(chan bool, 1)
	reaperCallbacks = make([]callback, 0)
	reaperWaitgroup = sync.WaitGroup{}
	reaperRegistrations = make([]string, 0)
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
			slog.Info("reaper: calling reap callback for '" + callback.name + "'")
			callback.callbackFunc()
		}
	}
}

func Callback(name string, callbackFunc func()) {
	reaperCallbacks = append(reaperCallbacks, callback{
		name:         name,
		callbackFunc: callbackFunc,
	})
}

func Register(name string) {
	if slices.Contains(reaperRegistrations, name) {
		slog.Warn("reaper: already registered '" + name + "'")
		return
	} else {
		reaperRegistrations = append(reaperRegistrations, name)
		reaperWaitgroup.Add(1)
		slog.Debug("reaper: registered '" + name + "'")
	}
}

func Done(name string) {
	if slices.Contains(reaperRegistrations, name) {
		reaperRegistrations = slices.DeleteFunc(reaperRegistrations, func(test string) bool {
			return test == name
		})

		slog.Debug("reaper: done: '" + name + "'")
		reaperWaitgroup.Done()
	} else {
		slog.Warn("reaper: already done or doesn't exist: '" + name + "'")
	}
}

func Wait() {
	reaperWaitgroup.Wait()
}
