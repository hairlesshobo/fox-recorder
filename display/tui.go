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
	"log/slog"
	"time"

	"fox-audio/display/custom"
	"fox-audio/display/theme"
	"fox-audio/model"
	"fox-audio/reaper"
	"fox-audio/util"

	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
)

//
// constants
//

const (
	StatusPaused       = 0
	StatusPlaying      = 1
	StatusRecording    = 2
	StatusStarting     = 3
	StatusShuttingDown = 4

	meterWidth = 4
)

//
// variables
//

var (
	meterSteps = []int{
		0, -1, -2, -3, -4, -6, -8,
		-10, -12, -15, -18, -21, -24, -27,
		-30, -36, -42, -48, -54, -60}

	levelColors = map[int]tcell.Color{
		0:    theme.Red,       // 124?
		-2:   theme.Pink,      // 124?
		-6:   theme.Yellow,    // 131?
		-18:  theme.Green,     // 142? 65? muted 71?
		-150: theme.SoftGreen, //tcell.Color120, 59? 60? 61? 66? 67? 68? 72?
	}
)

//
// types
//

type Tui struct {
	app             *cview.Application
	shutdownChannel chan bool

	errorCount int
	// sessionName           string
	// profileName           string
	// jackServerStatus      int    // 0 = not running, 1 = running, 2 = running with warnings, 3 = terminated
	// armedChannelCount     int
	// connectedChannelCount int
	// diskPerformancePct    float64

	appGrid    *cview.Grid
	meters     []*custom.LevelMeter
	metersGrid *cview.Grid

	tvLogs            *cview.TextView
	tvTransportStatus *custom.StatusText
	tvPosition        *custom.StatusText
	tvFormat          *custom.StatusText
	tvFileSize        *custom.StatusText
	tvErrorCount      *custom.StatusText

	meterDiskSpace *custom.StatusMeter
	meterBuffer    *custom.StatusMeter
	meterAudioLoad *custom.StatusMeter
}

//
// constructor
//

func NewTui() *Tui {
	tui := &Tui{
		shutdownChannel: make(chan bool, 1),
		errorCount:      0,
	}

	return tui
}

//
// lifecycle managment
//

func (tui *Tui) Initalize() {
	tui.app = cview.NewApplication()
	defer tui.app.HandlePanic()

	meterRowHeight := len(meterSteps) + 2

	tui.appGrid = cview.NewGrid()
	tui.appGrid.SetPadding(0, 0, 0, 0)
	tui.appGrid.SetColumns(-1, 60)
	tui.appGrid.SetRows(10, meterRowHeight, -1)
	tui.appGrid.SetBackgroundColor(cview.Styles.PrimitiveBackgroundColor)

	statusGrid := cview.NewGrid()
	statusGrid.SetBorder(true)
	statusGrid.SetPadding(0, 0, 1, 1)
	statusGrid.SetColumns(51, 55, -1)
	statusGrid.SetRows(1, 1, 1, 1, 1, 1, -1)
	statusGrid.SetBackgroundColor(cview.Styles.PrimitiveBackgroundColor)

	headerWidth := 16

	tui.tvTransportStatus = custom.NewStatusTextField(headerWidth, "Status", string(theme.RuneRecord)+" Recording")
	tui.tvTransportStatus.SetColor(theme.Red)
	tui.tvPosition = custom.NewStatusTextField(headerWidth, "Position", "00:00:00.000")
	tui.tvFormat = custom.NewStatusTextField(headerWidth, "Format", "Unknown")
	tui.tvFileSize = custom.NewStatusTextField(headerWidth, "Session Size", "0 bytes")
	tui.tvErrorCount = custom.NewStatusTextField(headerWidth, "Errors", "0")

	statusColOffset := 0
	statusGrid.AddItem(tui.tvTransportStatus.GetGrid(), 0, statusColOffset, 1, 1, 0, 0, false)
	statusGrid.AddItem(tui.tvPosition.GetGrid(), 1, statusColOffset, 1, 1, 0, 0, false)
	statusGrid.AddItem(tui.tvFormat.GetGrid(), 2, statusColOffset, 1, 1, 0, 0, false)
	statusGrid.AddItem(tui.tvFileSize.GetGrid(), 3, statusColOffset, 1, 1, 0, 0, false)
	statusGrid.AddItem(tui.tvErrorCount.GetGrid(), 4, statusColOffset, 1, 1, 0, 0, false)

	tui.meterDiskSpace = custom.NewStatusMeter(headerWidth, "Disk Space", 0, "%")
	tui.meterBuffer = custom.NewStatusMeter(headerWidth, "Buffer", 0, "%")
	tui.meterAudioLoad = custom.NewStatusMeter(headerWidth, "Audio Load", 0, "%")

	meterColOffset := 1
	statusGrid.AddItem(tui.meterDiskSpace.GetGrid(), 0, meterColOffset, 1, 1, 0, 0, false)
	statusGrid.AddItem(tui.meterBuffer.GetGrid(), 1, meterColOffset, 1, 1, 0, 0, false)
	statusGrid.AddItem(tui.meterAudioLoad.GetGrid(), 2, meterColOffset, 1, 1, 0, 0, false)
	tui.appGrid.AddItem(statusGrid, 0, 0, 1, 1, 0, 0, false)

	tui.metersGrid = cview.NewGrid()
	tui.metersGrid.SetBorder(false)
	tui.metersGrid.SetBorders(false)
	tui.metersGrid.SetPadding(0, 0, 0, 0)
	tui.metersGrid.SetColumns(-1)
	tui.appGrid.AddItem(tui.metersGrid, 1, 0, 1, 1, 0, 0, false)

	tui.tvLogs = cview.NewTextView()
	tui.tvLogs.SetPadding(0, 0, 0, 0)
	tui.appGrid.AddItem(tui.tvLogs, 2, 0, 1, 1, 0, 0, false)

	tui.app.SetRoot(tui.appGrid, true)
}

func (tui *Tui) Start() {
	reaper.Register("tui")

	go func() {
		defer tui.app.HandlePanic()

		// Capture user input
		tui.app.SetInputCapture(tui.eventHandler)

		if err := tui.app.Run(); err != nil {
			panic(err)
		}

		tui.shutdownChannel <- true
		reaper.Done("tui")
	}()

	go tui.excecuteLoop()
}

func (tui *Tui) Shutdown() {
	slog.Info("Shutting down TUI")
	tui.app.Stop()

	slog.Info("Waiting for TUI to shut down")
	tui.WaitForShutdown()
}

func (tui *Tui) IsShutdown() bool {
	return len(tui.shutdownChannel) > 0
}

func (tui *Tui) WaitForShutdown() {
	<-tui.shutdownChannel
}

//
// private functions
//

func (tui *Tui) eventHandler(event *tcell.EventKey) *tcell.EventKey {
	// Anything handled here will be executed on the main thread
	switch event.Key() {
	case tcell.KeyEsc:
	case tcell.KeyCtrlC:
		go reaper.Reap()
		return nil
	}

	return event
}

func (tui *Tui) excecuteLoop() {
	defer tui.app.HandlePanic()

	slog.Info("TUI loop started")

	for {
		if len(tui.shutdownChannel) > 0 {
			slog.Info("TUI shutting down")
			tui.app.QueueUpdateDraw(func() {})
			break
		}

		tui.app.QueueUpdateDraw(func() {})
		time.Sleep(25 * time.Millisecond)
	}

	fmt.Println("shutting down tui")
}

//
// status update functions
//

func (tui *Tui) SetTransportStatus(status int) {
	if status < 0 || status > 4 {
		panic("invalid status value provided: " + string(status))
	}

	var icon rune
	var color tcell.Color
	var transportStatus string

	if status == StatusPaused {
		icon = theme.RunePause
		color = theme.Blue
		transportStatus = "Paused"
	} else if status == StatusPlaying {
		icon = theme.RunePlay
		color = theme.Green
		transportStatus = "Playing"
	} else if status == StatusRecording {
		icon = theme.RuneRecord
		color = theme.Red
		transportStatus = "Recording"
	} else if status == StatusStarting {
		icon = theme.RuneClock
		color = theme.Yellow
		transportStatus = "Starting Audio Server"
	} else if status == StatusShuttingDown {
		icon = theme.RuneClock
		color = theme.Yellow
		transportStatus = "Shutting down"
	}

	tui.tvTransportStatus.SetCurrentValue(string(icon) + " " + transportStatus)
	tui.tvTransportStatus.SetColor(color)
}

func (tui *Tui) SetDuration(duration float64) {
	tui.tvPosition.SetCurrentValue(util.FormatDuration(duration))
}

func (tui *Tui) SetAudioFormat(format string) {
	tui.tvFormat.SetCurrentValue(format)
}

func (tui *Tui) SetSessionSize(size uint64) {
	tui.tvFileSize.SetCurrentValue(util.FormatSize(size))
}

func (tui *Tui) SetAudioLoad(percent int) {
	tui.meterAudioLoad.SetCurrentValue(percent)

	if percent <= 20 {
		tui.meterAudioLoad.SetColor(theme.Green)
	} else if percent <= 50 {
		tui.meterAudioLoad.SetColor(theme.Yellow)
	} else {
		tui.meterAudioLoad.SetColor(theme.Red)
	}
}

func (tui *Tui) UpdateSignalLevels(levels []*model.SignalLevel) {
	for i := range levels {
		level := levels[i]
		tui.meters[i].SetLevel(level.Instant)
	}
}

func (tui *Tui) SetDiskUsage(percent int) {
	tui.meterDiskSpace.SetCurrentValue(percent)

	if percent <= 20 {
		tui.meterDiskSpace.SetColor(theme.Green)
	} else if percent <= 50 {
		tui.meterDiskSpace.SetColor(theme.Yellow)
	} else {
		tui.meterDiskSpace.SetColor(theme.Red)
	}
}

func (tui *Tui) SetBufferUtilization(percent int) {
	tui.meterBuffer.SetCurrentValue(percent)

	if percent <= 50 {
		tui.meterBuffer.SetColor(theme.Green)
	} else if percent <= 75 {
		tui.meterBuffer.SetColor(theme.Yellow)
	} else {
		tui.meterBuffer.SetColor(theme.Red)
	}
}

func (tui *Tui) SetChannelCount(channelCount int) {
	levelColumns := make([]int, channelCount+2)
	levelColumns[0] = 5
	for i := range channelCount {
		levelColumns[i+1] = meterWidth
	}
	levelColumns[channelCount+1] = -1

	tui.metersGrid.SetColumns(levelColumns...)

	tui.meters = make([]*custom.LevelMeter, channelCount)

	meterStepLabel := cview.NewTextView()
	meterStepLabel.SetPadding(0, 0, 0, 0)

	// meterStepLabels := make([]string, len(meterSteps))
	meterStepLabel.Write([]byte(fmt.Sprintln()))
	for step := 0; step < len(meterSteps); step++ {
		meterStepLabel.Write([]byte(fmt.Sprintf("%3v\n", fmt.Sprintf("%d", meterSteps[step]))))
		// meterStepLabels = append(meterStepLabels, fmt.Sprintf("%3v", fmt.Sprintf("%d", meterSteps[step])))
	}
	tui.metersGrid.AddItem(meterStepLabel, 0, 0, 1, 1, 0, 0, false)

	for i := range channelCount {
		tui.meters[i] = custom.NewLevelMeter(meterSteps, levelColors)
		tui.meters[i].SetBorder(false)
		tui.meters[i].SetPadding(0, 0, 1, 1)
		tui.meters[i].SetMinLevel(-150)
		tui.meters[i].SetLevel(-99)
		tui.meters[i].SetChannelNumber(fmt.Sprintf("%d", i+1))
		tui.meters[i].ArmChannel(true)

		// TODO: this needs to be controlled externally

		// if (i >= 8 && i <= 16) || (i >= 19 && i <= 27) {
		// 	tui.meters[i].ArmChannel(true)
		// }

		if i%2 == 1 {
			tui.meters[i].SetBackgroundColor(theme.MeterAlternateBackground)
		}

		tui.metersGrid.AddItem(tui.meters[i], 0, i+1, 1, 1, 0, 0, false)
	}
}

func (tui *Tui) WriteLog(message string) {
	tui.tvLogs.Write([]byte(fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), message)))
}

func (tui *Tui) IncrementErrorCount() {
	tui.errorCount++
	tui.tvErrorCount.SetCurrentValue(fmt.Sprintf("%d", tui.errorCount))

	if tui.errorCount > 0 {
		tui.tvErrorCount.SetColor(theme.Red)
	}
}
