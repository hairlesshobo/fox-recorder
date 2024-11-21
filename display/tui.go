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
	"time"

	"fox-audio/custom"
	"fox-audio/shared"

	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
)

var (
	meterWidth = 4
	// channels   = 32
	meterSteps = []int{
		0, -1, -2, -3, -4, -6, -8,
		-10, -12, -15, -18, -21, -24, -27,
		-30, -36, -42, -48, -54, -60}

	levelColors = map[int]tcell.Color{
		-1:   tcell.ColorDarkRed,
		-6:   tcell.Color130,
		-18:  tcell.ColorGreen,
		-150: tcell.Color120,
	}
)

type Tui struct {
	// statusUpdateChannel chan int
	channelCount    int
	app             *cview.Application
	shutdownChannel chan bool

	sessionName           string
	profileName           string
	jackServerStatus      int // 0 = not running, 1 = running, 2 = running with warnings, 3 = terminated
	transportStatus       string
	bitDepth              int
	sampleRate            int
	armedChannelCount     int
	connectedChannelCount int
	diskTotal             int64
	diskUsed              int64
	diskRremainingTime    int64
	recordingDuration     int64
	bufferUsagePct        float64
	diskPerformancePct    float64

	meters            []*custom.LevelMeter
	tvLogs            *cview.TextView
	tvTransportStatus *cview.TextView
	tvPosition        *cview.TextView
}

func NewTui(channels int) *Tui {
	tui := &Tui{
		channelCount:    channels,
		shutdownChannel: make(chan bool, 1),
	}

	return tui
}

func (tui *Tui) addStatusTextField(grid *cview.Grid, row int, name string, initialValue string) *cview.TextView {
	header := cview.NewTextView()
	header.SetTextAlign(cview.AlignRight)
	header.Write([]byte(fmt.Sprintf("%s: ", name)))
	grid.AddItem(header, row, 0, 1, 1, 0, 0, false)

	valueTextView := cview.NewTextView()
	valueTextView.Write([]byte(initialValue))
	grid.AddItem(valueTextView, row, 1, 1, 1, 0, 0, false)

	return valueTextView
}

func (tui *Tui) Initalize() {
	tui.app = cview.NewApplication()
	defer tui.app.HandlePanic()

	meterRowHeight := len(meterSteps) + 3

	grid := cview.NewGrid()
	grid.SetPadding(0, 0, 0, 0)
	grid.SetColumns(-1, 30)
	grid.SetRows(10, meterRowHeight, -1)
	grid.SetBackgroundColor(cview.Styles.PrimitiveBackgroundColor)

	statusGrid := cview.NewGrid()
	statusGrid.SetBorder(true)
	statusGrid.SetPadding(0, 0, 1, 1)
	statusGrid.SetColumns(12, -1, -1)
	statusGrid.SetRows(1, 1, 1, 1, 1, 1, -1)
	statusGrid.SetBackgroundColor(cview.Styles.PrimitiveBackgroundColor)

	// todo put this in struct
	tui.tvTransportStatus = tui.addStatusTextField(statusGrid, 0, "Status", "Recording")
	tui.tvPosition = tui.addStatusTextField(statusGrid, 1, "Position", "00:00:00.000")

	grid.AddItem(statusGrid, 0, 0, 1, 1, 0, 0, false)

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

	tui.tvLogs = cview.NewTextView()
	tui.tvLogs.SetPadding(0, 0, 0, 0)
	grid.AddItem(tui.tvLogs, 2, 0, 1, 1, 0, 0, false)

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

	// ready <- true

	tui.app.SetRoot(grid, true)
}

func (tui *Tui) WriteLog(message string) {
	tui.tvLogs.Write([]byte(fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), message)))
}

func (tui *Tui) excecuteLoop() {
	defer tui.app.HandlePanic()

	tui.WriteLog(("TUI loop started"))

	for {
		if len(tui.shutdownChannel) > 0 {
			tui.WriteLog("TUI shutting down")
			tui.app.QueueUpdateDraw(func() {})
			break
		}

		tui.app.QueueUpdateDraw(func() {})
		time.Sleep(25 * time.Millisecond)
	}

	fmt.Println("shutting down tui")
}

func (tui *Tui) Start() {
	go func() {
		defer tui.app.HandlePanic()

		if err := tui.app.Run(); err != nil {
			panic(err)
		}

		fmt.Println("requested exit")
		tui.shutdownChannel <- true
	}()

	go tui.excecuteLoop()
}

func (tui *Tui) UpdateSignalLevels(levels []*shared.SignalLevel) {
	for i := range levels {
		level := levels[i]
		tui.meters[i].SetLevel(level.Instant)
	}
}

func (tui *Tui) IsShutdown() bool {
	return len(tui.shutdownChannel) > 0
}

func (tui *Tui) WaitForShutdown() {
	<-tui.shutdownChannel
}
