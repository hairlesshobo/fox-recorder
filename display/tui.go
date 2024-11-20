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
package display

import (
	"fmt"
	// "math/rand/v2"
	// "strings"
	"time"

	"fox-audio/custom"
	"fox-audio/shared"

	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
)

var meterWidth = 5

// var channels = 32
var meterSteps = []int{
	0, -1, -2, -3, -4, -6, -8,
	-10, -12, -15, -18, -21, -24, -27,
	-30, -36, -42, -48, -54, -60}

var levelColors = map[int]tcell.Color{
	-1:   tcell.ColorDarkRed,
	-6:   tcell.Color130,
	-18:  tcell.ColorGreen,
	-150: tcell.Color120,
}

type Tui struct {
	// statusUpdateChannel chan int
	channelCount        int
	app                 *cview.Application
	signalUpdateChannel chan []*shared.SignalLevel
	shutdownChannel     chan bool
	statusView          *cview.TextView
	logView             *cview.TextView
	meters              []*custom.LevelMeter
}

func NewTui(channels int) *Tui {
	tui := &Tui{
		channelCount:        channels,
		shutdownChannel:     make(chan bool),
		signalUpdateChannel: make(chan []*shared.SignalLevel),
	}

	return tui
}

func (tui *Tui) Initalize(ready chan bool) {
	tui.app = cview.NewApplication()
	defer tui.app.HandlePanic()

	meterRowHeight := len(meterSteps) + 3

	grid := cview.NewGrid()
	grid.SetPadding(0, 0, 0, 0)
	grid.SetColumns(-1)
	grid.SetRows(10, meterRowHeight, -1)
	grid.SetBackgroundColor(cview.Styles.PrimitiveBackgroundColor)

	tui.statusView = cview.NewTextView()
	tui.statusView.SetBorder(true)
	tui.statusView.SetPadding(0, 0, 0, 0)
	tui.statusView.Write([]byte("Status    : Recording\n"))
	tui.statusView.Write([]byte("Elapsed   : 00:00:00.000\n"))
	grid.AddItem(tui.statusView, 0, 0, 1, 1, 0, 0, false)

	levelColumns := make([]int, tui.channelCount+2)
	levelColumns[0] = 5
	for i := range tui.channelCount {
		levelColumns[i+1] = meterWidth
	}
	levelColumns[tui.channelCount+1] = -1

	levelsGrid := cview.NewGrid()
	levelsGrid.SetBorder(false)
	levelsGrid.SetBorders(false)
	levelsGrid.SetPadding(0, 0, 0, 0)
	levelsGrid.SetColumns(levelColumns...)
	// levelsGrid.SetRows(2, 0, 1)
	levelsGrid.SetRows(0)
	grid.AddItem(levelsGrid, 1, 0, 1, 1, 0, 0, false)

	tui.logView = cview.NewTextView()
	tui.logView.SetPadding(0, 0, 0, 0)
	tui.logView.SetText("this is the logs view\n")
	grid.AddItem(tui.logView, 2, 0, 1, 1, 0, 0, false)

	tui.meters = make([]*custom.LevelMeter, tui.channelCount)

	meterStepLabel := cview.NewTextView()
	meterStepLabel.SetPadding(0, 0, 0, 0)

	// meterStepLabels := make([]string, len(meterSteps))
	meterStepLabel.Write([]byte(fmt.Sprintln()))
	for step := 0; step < len(meterSteps); step++ {
		meterStepLabel.Write([]byte(fmt.Sprintf("%3v\n", fmt.Sprintf("%d", meterSteps[step]))))
		// meterStepLabels = append(meterStepLabels, fmt.Sprintf("%3v", fmt.Sprintf("%d", meterSteps[step])))
	}
	levelsGrid.AddItem(meterStepLabel, 0, 0, 1, 1, 0, 0, false)

	for i := range tui.channelCount {
		tui.meters[i] = custom.NewLevelMeter(meterSteps, levelColors)
		tui.meters[i].SetBorder(false)
		tui.meters[i].SetPadding(0, 0, 1, 1)
		tui.meters[i].SetMinLevel(-150)
		tui.meters[i].SetLevel(-99)
		tui.meters[i].SetChannelNumber(fmt.Sprintf("%d", i+1))
		if i%2 == 1 {
			tui.meters[i].SetBackgroundColor(tcell.Color233)
		}

		levelsGrid.AddItem(tui.meters[i], 0, i+1, 1, 1, 0, 0, false)
	}

	// go func() {
	// 	t := time.NewTicker(100 * time.Millisecond)
	// 	for range t.C {
	// 		for i := range tui.channelCount {
	// 			newLevel := rand.IntN(24) * (-1)
	// 			meters[i].SetLevel(newLevel)
	// 			if newLevel > meters[i].GetLongTermMaxLevel() {
	// 				meters[i].SetLongTermMaxLevel(newLevel)
	// 			}
	// 		}

	// 		// Queue draw
	// 		app.QueueUpdateDraw(func() {})
	// 	}
	// }()

	ready <- true

	tui.app.SetRoot(grid, true)
	if err := tui.app.Run(); err != nil {
		panic(err)
	}
}

func (tui *Tui) WriteLog(message string) {
	tui.logView.Write([]byte(fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), message)))
}

func (tui *Tui) excecuteLoop() {
	tui.WriteLog(("TUI loop started"))
out:
	for {
		select {
		// check for shutdown signal
		case shutdown := <-tui.shutdownChannel:
			if shutdown {
				tui.WriteLog("TUI shutting down")
				tui.app.QueueUpdateDraw(func() {})
				break out
			}
		case levels := <-tui.signalUpdateChannel:
			for i := range levels {
				level := levels[i]
				tui.meters[i].SetLevel(level.Instant)
			}
		default:
			// continue processing here
			// // Queue draw
		}

		tui.app.QueueUpdateDraw(func() {})
		time.Sleep(25 * time.Millisecond)
	}
}

func (tui *Tui) Start() {
	go tui.excecuteLoop()
}

func (tui *Tui) UpdateSignalLevels(levels []*shared.SignalLevel) {
	tui.signalUpdateChannel <- levels
}
