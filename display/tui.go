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
	layoutMeterWidth            = 4
	layoutStatusItemHeaderWidth = 18
	layoutStatusColumnIndex     = 0
	layoutMeterColumnIndex      = 1
	layoutStatusGridLeftWidth   = 51
	layoutStatusGridRightWidth  = 55

	layoutOutputFileColumnWidth = 45
	layoutOutputFilePortsWidth  = 8
	layoutOutputFileSizeWidth   = 11
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
		0:    theme.Red,
		-2:   theme.Pink,
		-6:   theme.Yellow,
		-18:  theme.Green,
		-150: theme.SoftGreen,
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
	// jackServerStatus      int    // 0 = not running, 1 = running, 2 = running with warnings, 3 = terminated
	// armedChannelCount     int
	// connectedChannelCount int

	gridApp            *cview.Grid
	gridLevelMeters    *cview.Grid
	gridOutputFiles    *cview.Grid
	elementLevelMeters []*custom.LevelMeter
	elementOutputFiles []*custom.OutputFileField

	tvLogs            *cview.TextView
	tvTransportStatus *custom.StatusText
	tvPosition        *custom.StatusText
	tvFormat          *custom.StatusText
	tvFileSize        *custom.StatusText
	tvErrorCount      *custom.StatusText
	tvProfileName     *custom.StatusText
	tvTakeName        *custom.StatusText
	tvDirectory       *custom.StatusText

	statusMeterDiskUsed        *custom.StatusMeter
	statusMeterBufferUsed      *custom.StatusMeter
	statusMeterCycleBufferUsed *custom.StatusMeter
	statusMeterAudioLoad       *custom.StatusMeter
	statusMeterDiskLoad        *custom.StatusMeter
}

//
// constructor
//

func NewTui() *Tui {
	tui := &Tui{
		shutdownChannel:    make(chan bool, 1),
		errorCount:         0,
		elementLevelMeters: make([]*custom.LevelMeter, 0),
		elementOutputFiles: make([]*custom.OutputFileField, 0),
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

	statusRowCount := 10
	statusRows := make([]int, statusRowCount)
	for i := range statusRowCount {
		statusRows[i] = 1
	}

	//
	// main application grid
	tui.gridApp = cview.NewGrid()
	tui.gridApp.SetPadding(0, 0, 0, 0)
	tui.gridApp.SetColumns(-1, layoutOutputFileColumnWidth)
	tui.gridApp.SetBorders(true)
	tui.gridApp.SetBordersColor(theme.BorderColor)
	tui.gridApp.SetRows(statusRowCount, meterRowHeight, -1)
	tui.gridApp.SetBackgroundColor(cview.Styles.PrimitiveBackgroundColor)

	//
	// grid for the output files list
	tui.gridOutputFiles = cview.NewGrid()
	tui.gridOutputFiles.SetPadding(0, 0, 0, 0)
	tui.gridOutputFiles.SetColumns(-1)
	tui.gridOutputFiles.SetRows(-1)
	tui.gridOutputFiles.SetBackgroundColor(cview.Styles.PrimitiveBackgroundColor)

	tui.gridApp.AddItem(tui.gridOutputFiles, 0, 1, 3, 1, 0, 0, false)

	//
	// grid for the status meters
	gridStatusMeters := cview.NewGrid()
	gridStatusMeters.SetPadding(0, 0, 1, 1)
	gridStatusMeters.SetColumns(layoutStatusGridLeftWidth, layoutStatusGridRightWidth, -1)
	gridStatusMeters.SetRows(statusRows...)
	gridStatusMeters.SetBackgroundColor(cview.Styles.PrimitiveBackgroundColor)

	// text status fields
	tui.tvTransportStatus = custom.NewStatusTextField(layoutStatusItemHeaderWidth, "Status", string(theme.RuneRecord)+" Recording")
	tui.tvTransportStatus.SetColor(theme.Red)
	tui.tvPosition = custom.NewStatusTextField(layoutStatusItemHeaderWidth, "Position", "00:00:00.000")
	tui.tvFormat = custom.NewStatusTextField(layoutStatusItemHeaderWidth, "Format", "Unknown")
	tui.tvFileSize = custom.NewStatusTextField(layoutStatusItemHeaderWidth, "Session Size", "0 bytes")
	tui.tvErrorCount = custom.NewStatusTextField(layoutStatusItemHeaderWidth, "Errors", "0")
	tui.tvProfileName = custom.NewStatusTextField(layoutStatusItemHeaderWidth, "Profile", "")
	tui.tvTakeName = custom.NewStatusTextField(layoutStatusItemHeaderWidth, "Take", "")
	tui.tvDirectory = custom.NewStatusTextField(layoutStatusItemHeaderWidth, "Directory", "")

	gridStatusMeters.AddItem(tui.tvTransportStatus.GetGrid(), 0, layoutStatusColumnIndex, 1, 1, 0, 0, false)
	gridStatusMeters.AddItem(tui.tvPosition.GetGrid(), 1, layoutStatusColumnIndex, 1, 1, 0, 0, false)
	gridStatusMeters.AddItem(tui.tvFormat.GetGrid(), 2, layoutStatusColumnIndex, 1, 1, 0, 0, false)
	gridStatusMeters.AddItem(tui.tvFileSize.GetGrid(), 3, layoutStatusColumnIndex, 1, 1, 0, 0, false)
	gridStatusMeters.AddItem(tui.tvErrorCount.GetGrid(), 4, layoutStatusColumnIndex, 1, 1, 0, 0, false)
	gridStatusMeters.AddItem(tui.tvProfileName.GetGrid(), 5, layoutStatusColumnIndex, 1, 1, 0, 0, false)
	gridStatusMeters.AddItem(tui.tvTakeName.GetGrid(), 6, layoutStatusColumnIndex, 1, 1, 0, 0, false)
	gridStatusMeters.AddItem(tui.tvDirectory.GetGrid(), statusRowCount-1, layoutStatusColumnIndex, 1, 2, 0, 0, false)

	// progress bar status meters
	tui.statusMeterDiskUsed = custom.NewStatusMeter(layoutStatusItemHeaderWidth, "Disk Space", 0, "%")
	tui.statusMeterDiskLoad = custom.NewStatusMeter(layoutStatusItemHeaderWidth, "Disk Load", 0, "%")
	tui.statusMeterAudioLoad = custom.NewStatusMeter(layoutStatusItemHeaderWidth, "Audio Load", 0, "%")
	tui.statusMeterBufferUsed = custom.NewStatusMeter(layoutStatusItemHeaderWidth, "Buffer", 0, "%")
	tui.statusMeterCycleBufferUsed = custom.NewStatusMeter(layoutStatusItemHeaderWidth, "Cycle Buffer", 0, "%")

	gridStatusMeters.AddItem(tui.statusMeterDiskUsed.GetGrid(), 0, layoutMeterColumnIndex, 1, 1, 0, 0, false)
	gridStatusMeters.AddItem(tui.statusMeterDiskLoad.GetGrid(), 1, layoutMeterColumnIndex, 1, 1, 0, 0, false)
	gridStatusMeters.AddItem(tui.statusMeterAudioLoad.GetGrid(), 2, layoutMeterColumnIndex, 1, 1, 0, 0, false)
	gridStatusMeters.AddItem(tui.statusMeterBufferUsed.GetGrid(), 3, layoutMeterColumnIndex, 1, 1, 0, 0, false)
	gridStatusMeters.AddItem(tui.statusMeterCycleBufferUsed.GetGrid(), 4, layoutMeterColumnIndex, 1, 1, 0, 0, false)

	tui.gridApp.AddItem(gridStatusMeters, 0, 0, 1, 1, 0, 0, false)

	//
	// grid for the level meters
	tui.gridLevelMeters = cview.NewGrid()
	tui.gridLevelMeters.SetPadding(0, 0, 0, 0)
	tui.gridLevelMeters.SetColumns(-1)

	tui.gridApp.AddItem(tui.gridLevelMeters, 1, 0, 1, 1, 0, 0, false)

	//
	// grid for the log output view
	tui.tvLogs = cview.NewTextView()
	tui.tvLogs.SetPadding(0, 0, 0, 0)
	tui.tvLogs.SetDynamicColors(true)

	tui.gridApp.AddItem(tui.tvLogs, 2, 0, 1, 1, 0, 0, true)

	tui.app.SetRoot(tui.gridApp, true)
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
	slog.Debug("Shutting down TUI")
	tui.app.Stop()

	slog.Debug("Waiting for TUI to shut down")
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

	slog.Debug("TUI loop started")

	for {
		if len(tui.shutdownChannel) > 0 {
			slog.Info("TUI shutting down")
			tui.app.QueueUpdateDraw(func() {})
			break
		}

		tui.app.QueueUpdateDraw(func() {})
		time.Sleep(50 * time.Millisecond)
	}

	// fmt.Println("shutting down tui")
}

func (tui *Tui) updateMeter(meter *custom.StatusMeter, value, warnPct, cautionPct int) {
	color := tcell.ColorDefault

	if value <= warnPct {
		color = theme.Green
	} else if value <= cautionPct {
		color = theme.Yellow
	} else {
		color = theme.Red
	}

	meter.SetCurrentValue(value)
	meter.SetColor(color)
}

//
// status update functions
//

func (tui *Tui) SetTransportStatus(status Status) {
	if status < 0 || status > 5 {
		panic("invalid status value provided: " + string(status))
	}

	var icon rune
	var color tcell.Color
	var transportStatus string

	if status == StatusPaused {
		icon = theme.RunePause
		color = theme.Blue
	} else if status == StatusPlaying {
		icon = theme.RunePlay
		color = theme.Green
	} else if status == StatusRecording {
		icon = theme.RuneRecord
		color = theme.Red
	} else if status == StatusStarting {
		icon = theme.RuneClock
		color = theme.Yellow
	} else if status == StatusShuttingDown {
		icon = theme.RuneClock
		color = theme.Yellow
	} else if status == StatusFailed {
		icon = theme.RuneFailed
		color = theme.Red
	}

	transportStatus = statusNames[status]

	tui.tvTransportStatus.SetCurrentValue(string(icon) + " " + transportStatus)
	tui.tvTransportStatus.SetColor(color)
}

func (tui *Tui) SetDuration(duration float64) {
	tui.tvPosition.SetCurrentValue(util.FormatDuration(duration))
}

func (tui *Tui) SetAudioFormat(format string) {
	tui.tvFormat.SetCurrentValue(format)
}

func (tui *Tui) SetProfileName(value string) {
	tui.tvProfileName.SetCurrentValue(value)
}

func (tui *Tui) SetTakeName(value string) {
	tui.tvTakeName.SetCurrentValue(value)
}

func (tui *Tui) SetDirectory(value string) {
	tui.tvDirectory.SetCurrentValue(value)
}

func (tui *Tui) SetSessionSize(size uint64) {
	tui.tvFileSize.SetCurrentValue(util.FormatSize(size))
}

func (tui *Tui) IncrementErrorCount() {
	tui.errorCount++
	tui.tvErrorCount.SetCurrentValue(fmt.Sprintf("%d", tui.errorCount))

	if tui.errorCount > 0 {
		tui.tvErrorCount.SetColor(theme.Red)
	}
}

//
// channel strips
//

func (tui *Tui) UpdateSignalLevels(levels []model.SignalLevel) {
	for i := range levels {
		level := levels[i]
		tui.elementLevelMeters[i].SetLevel(level.Instant)
	}
}

func (tui *Tui) SetChannelArmStatus(channel int, armed bool) {
	tui.elementLevelMeters[channel].ArmChannel(armed)
}

func (tui *Tui) SetOutputFiles(outputFiles []model.UiOutputFile) {
	fileCount := len(outputFiles)
	tui.elementOutputFiles = make([]*custom.OutputFileField, fileCount)

	outputFileRows := make([]int, fileCount+1)
	for i := range fileCount {
		outputFileRows[i] = 1
	}
	outputFileRows[fileCount] = -1

	tui.gridOutputFiles.SetRows(outputFileRows...)

	// loop through and create a new output file ui item for each output file
	for i, outputFile := range outputFiles {
		outputFileField := custom.NewOutputFileField(layoutOutputFilePortsWidth, layoutOutputFileSizeWidth, outputFile)
		tui.elementOutputFiles[i] = outputFileField
		tui.gridOutputFiles.AddItem(outputFileField.GetGrid(), i, 0, 1, 1, 0, 0, false)
	}
}

func (tui *Tui) UpdateOutputFileSizes(sizes []uint64) {
	for i, size := range sizes {
		tui.elementOutputFiles[i].SetSize(size)
	}
}

func (tui *Tui) SetChannelCount(channelCount int) {
	tui.elementLevelMeters = make([]*custom.LevelMeter, channelCount)

	levelColumns := make([]int, channelCount+2)
	levelColumns[0] = 5
	for i := range channelCount {
		levelColumns[i+1] = layoutMeterWidth
	}
	levelColumns[channelCount+1] = -1

	tui.gridLevelMeters.SetColumns(levelColumns...)

	meterStepLabel := cview.NewTextView()
	meterStepLabel.SetPadding(0, 0, 0, 0)

	meterStepLabel.Write([]byte(fmt.Sprintln()))
	for step := 0; step < len(meterSteps); step++ {
		meterStepLabel.Write([]byte(fmt.Sprintf("%3v\n", fmt.Sprintf("%d", meterSteps[step]))))
	}
	tui.gridLevelMeters.AddItem(meterStepLabel, 0, 0, 1, 1, 0, 0, false)

	for i := range channelCount {
		tui.elementLevelMeters[i] = custom.NewLevelMeter(meterSteps, levelColors)
		tui.elementLevelMeters[i].SetBorder(false)
		tui.elementLevelMeters[i].SetPadding(0, 0, 1, 1)
		tui.elementLevelMeters[i].SetMinLevel(-150)
		tui.elementLevelMeters[i].SetLevel(-99)
		tui.elementLevelMeters[i].SetChannelNumber(fmt.Sprintf("%d", i+1))
		tui.elementLevelMeters[i].ArmChannel(false)

		if i%2 == 1 {
			tui.elementLevelMeters[i].SetBackgroundColor(theme.LevelMeterAlternateBackgroundColor)
		}

		tui.gridLevelMeters.AddItem(tui.elementLevelMeters[i], 0, i+1, 1, 1, 0, 0, false)
	}
}

//
// logging
//

func (tui *Tui) WriteLevelLog(level slog.Level, message string) {
	color := "-"

	if level == slog.LevelWarn {
		color = "#" + theme.YellowRGB // "yellow"
	} else if level == slog.LevelError {
		color = "#" + theme.RedRGB + "::b"
	} else if level == slog.LevelDebug {
		color = "#" + theme.GrayRGB
	}

	tui.tvLogs.Write([]byte(fmt.Sprintf("[%s][%s[] [%s[] %s[-:-:-]\n", color, time.Now().Format("2006-01-02 15:04:05"), level.String(), message)))
}

//
// status meters
//

func (tui *Tui) SetAudioLoad(percent int) {
	tui.updateMeter(tui.statusMeterAudioLoad, percent, 20, 50)
}

func (tui *Tui) SetDiskUsage(percent int) {
	tui.updateMeter(tui.statusMeterDiskUsed, percent, 20, 50)
}

func (tui *Tui) SetBufferUtilization(percent int) {
	tui.updateMeter(tui.statusMeterBufferUsed, percent, 50, 75)
}

func (tui *Tui) SetDiskLoad(percent int) {
	tui.updateMeter(tui.statusMeterDiskLoad, percent, 50, 75)
}

func (tui *Tui) SetCycleBuffer(percent int) {
	tui.updateMeter(tui.statusMeterCycleBufferUsed, percent, 20, 50)
}
