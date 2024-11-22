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

type StatusMeter struct {
	showValue  bool
	unit       string
	grid       *cview.Grid
	headerView *cview.TextView
	meterView  *cview.ProgressBar
	valueView  *cview.TextView
}

func NewStatusMeter(headerWidth int, name string, initialValue int, unit string) *StatusMeter {
	meter := StatusMeter{
		grid:      cview.NewGrid(),
		unit:      unit,
		showValue: true,
	}

	columns := []int{headerWidth, -1}

	if meter.showValue {
		columns = append(columns, 6)
	}

	meter.grid.SetPadding(0, 0, 0, 0)
	meter.grid.SetColumns(columns...)
	meter.grid.SetRows(1)

	// TODO: make value view conditional
	meter.headerView = cview.NewTextView()
	meter.headerView.SetTextAlign(cview.AlignRight)
	meter.SetHeader(name)
	meter.grid.AddItem(meter.headerView, 0, 0, 1, 1, 0, 0, false)

	meter.meterView = cview.NewProgressBar()
	meter.meterView.SetFilledRune(rune(9607))
	meter.meterView.SetEmptyRune(rune(9617))
	meter.meterView.SetEmptyColor(tcell.Color242)
	meter.grid.AddItem(meter.meterView, 0, 1, 1, 1, 0, 0, false)

	if meter.showValue {
		meter.valueView = cview.NewTextView()
		meter.valueView.SetPadding(0, 0, 1, 0)
		meter.grid.AddItem(meter.valueView, 0, 2, 1, 1, 0, 0, false)
	}

	meter.SetCurrentValue(initialValue)

	return &meter
}

func (meter *StatusMeter) SetHeader(value string) {
	meter.headerView.Write([]byte(fmt.Sprintf("%s: ", value)))
}

func (meter *StatusMeter) SetCurrentValue(value int) {
	meter.meterView.SetProgress(value)
	meter.valueView.Clear()

	if meter.showValue {
		meter.valueView.Write([]byte(fmt.Sprintf("%d %s", value, meter.unit)))
	}
}

func (meter *StatusMeter) GetGrid() *cview.Grid {
	return meter.grid
}
