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
package util

import (
	"fox-audio/reaper"
	"time"
)

func ProcessOnInterval(milliseconds int, process func()) {
	go func() {
		reaper.Register()

		process()

		t := time.NewTicker(time.Duration(milliseconds) * time.Millisecond)

		for range t.C {
			if reaper.Reaped() {
				break
			}

			process()
		}

		reaper.Done()
	}()
}
