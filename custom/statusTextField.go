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
package custom

import (
	"fmt"

	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
)

type StatusText struct {
	grid       *cview.Grid
	color      tcell.Color
	headerView *cview.TextView
	valueView  *cview.TextView
}

func NewStatusTextField(headerWidth int, name string, initialValue string) *StatusText {
	field := StatusText{
		grid: cview.NewGrid(),
	}

	field.grid.SetPadding(0, 0, 0, 0)
	field.grid.SetColumns(headerWidth, -1)
	field.grid.SetRows(1)

	field.headerView = cview.NewTextView()
	field.headerView.SetTextAlign(cview.AlignRight)
	field.SetHeader(name)
	field.grid.AddItem(field.headerView, 0, 0, 1, 1, 0, 0, false)

	field.valueView = cview.NewTextView()
	field.valueView.Write([]byte(initialValue))
	field.grid.AddItem(field.valueView, 0, 1, 1, 1, 0, 0, false)

	field.SetCurrentValue(initialValue)

	return &field
}

func (field *StatusText) SetHeader(value string) {
	field.headerView.Write([]byte(fmt.Sprintf("%s: ", value)))
}

func (field *StatusText) SetCurrentValue(value string) {
	field.valueView.Clear()
	field.valueView.Write([]byte(value))
}

func (field *StatusText) SetColor(color tcell.Color) {
	field.color = color
	field.valueView.SetTextColor(color)
}

func (field *StatusText) GetGrid() *cview.Grid {
	return field.grid
}
