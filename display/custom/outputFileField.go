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
	"strings"

	"fox-audio/model"
	"fox-audio/util"

	"code.rocketnine.space/tslocum/cview"
)

type OutputFileField struct {
	grid      *cview.Grid
	portsView *cview.TextView
	nameView  *cview.TextView
	sizeView  *cview.TextView
}

func NewOutputFileField(portsWidth int, sizeWidth int, outputFile model.UiOutputFile) *OutputFileField {
	field := OutputFileField{
		grid: cview.NewGrid(),
	}

	field.grid.SetPadding(0, 0, 0, 0)
	field.grid.SetColumns(portsWidth, -1, sizeWidth)
	field.grid.SetRows(1)

	field.portsView = cview.NewTextView()
	field.portsView.SetTextAlign(cview.AlignRight)
	field.grid.AddItem(field.portsView, 0, 0, 1, 1, 0, 0, false)

	field.nameView = cview.NewTextView()
	field.nameView.SetPadding(0, 0, 2, 2)
	field.grid.AddItem(field.nameView, 0, 1, 1, 1, 0, 0, false)

	field.sizeView = cview.NewTextView()
	field.sizeView.SetTextAlign(cview.AlignRight)
	field.grid.AddItem(field.sizeView, 0, 2, 1, 1, 0, 0, false)

	field.SetPorts(outputFile.Ports)
	field.SetName(outputFile.Name)
	field.SetSize(outputFile.Size)

	return &field
}

func (field *OutputFileField) SetPorts(values []string) {
	field.portsView.Clear()
	field.portsView.Write([]byte(strings.Join(values, ", ")))
}

func (field *OutputFileField) SetName(value string) {
	field.nameView.Clear()
	field.nameView.Write([]byte(value))
}

func (field *OutputFileField) SetSize(size uint64) {
	field.sizeView.Clear()
	field.sizeView.Write([]byte(util.FormatSize(size)))
}

func (field *OutputFileField) GetGrid() *cview.Grid {
	return field.grid
}
