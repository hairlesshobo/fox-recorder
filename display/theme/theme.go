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
package theme

import (
	"github.com/gdamore/tcell/v2"
)

const (
	Blue         = tcell.ColorBlue
	BlueRGB      = "0000FF"
	Green        = tcell.Color71
	GreenRGB     = "5FAF5F"
	Pink         = tcell.Color131
	PinkRGB      = "AF5F5F"
	Red          = tcell.Color124
	RedRGB       = "AF0000"
	SoftGreen    = tcell.Color72
	SoftGreenRGB = "5FAF87"
	Yellow       = tcell.Color142
	YellowRGB    = "AFAF00"
	Gray         = tcell.ColorGray
	GrayRGB      = "808080"

	BorderColor = tcell.Color243

	LevelMeterAlternateBackgroundColor = tcell.Color233
	LevelMeterDisarmdFillColor         = tcell.Color242
)

// 0:    theme.Red, // 124?
// -2:   tcell.Color131, // 124?
// -6:   tcell.Color142, // 131?
// -18:  tcell.Color71,  // 142? 65? muted 71?
// -150: tcell.Color72,  //tcell.Color120, 59? 60? 61? 66? 67? 68? 72?

const (
	RuneClock       = rune(9201) // ⏱
	RunePause       = rune(9208) // ⏸
	RunePausePlay   = rune(9199) // ⏯
	RunePlay        = rune(9205) // ⏵  -- alternate: rune(9654)
	RuneRecord      = rune(9210) // ⏺  -- alternate: rune(9679)
	RuneSkipBack    = rune(9198) // ⏮
	RuneSkipForward = rune(9197) // ⏭
	RuneStop        = rune(9209) // ⏹  -- alternate: rune(9635)

	RuneFailed = rune(9932) // ⛌
	// 10004 ✔
	// 10008 ✘

	// mono/stereo possible indicators
	// 9900 ⚬
	// 9901 ⚭
	// 9902 ⚮
	// 9903 ⚯
)
