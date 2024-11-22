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

	"fox-audio/custom"
	"fox-audio/model"
	"fox-audio/reaper"
	"fox-audio/theme"

	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
)

const (
	runeStop        = rune(9209) // ⏹  -- alternate: rune(9635)
	runeRecord      = rune(9210) // ⏺  -- alternate: rune(9679)
	runePlay        = rune(9205) // ⏵  -- alternate: rune(9654)
	runePause       = rune(9208) // ⏸
	runePausePlay   = rune(9199) // ⏯
	runeSkipBack    = rune(9198) // ⏮
	runeSkipForward = rune(9197) // ⏭
	runeClock       = rune(9201) // ⏱

	StatusPaused    = 0
	StatusPlaying   = 1
	StatusRecording = 2

	meterAlternateBackground = tcell.Color233
	meterWidth               = 4
)

var (

	// channels   = 32
	meterSteps = []int{
		0, -1, -2, -3, -4, -6, -8,
		-10, -12, -15, -18, -21, -24, -27,
		-30, -36, -42, -48, -54, -60}

	levelColors = map[int]tcell.Color{
		// -1:   tcell.ColorDarkRed, // 124?
		// -6:   tcell.Color130,
		// -18:  tcell.ColorGreen, // 142? 65? muted 71?
		// -150: tcell.Color72,    //tcell.Color120, 59? 60? 61? 66? 67? 68? 72?

		0:    theme.Red,       // 124?
		-2:   theme.Pink,      // 124?
		-6:   theme.Yellow,    // 131?
		-18:  theme.Green,     // 142? 65? muted 71?
		-150: theme.SoftGreen, //tcell.Color120, 59? 60? 61? 66? 67? 68? 72?
	}
)

type Tui struct {
	// statusUpdateChannel chan int
	// channelCount    int
	app             *cview.Application
	shutdownChannel chan bool

	// sessionName           string
	// profileName           string
	// jackServerStatus      int    // 0 = not running, 1 = running, 2 = running with warnings, 3 = terminated
	// transportStatus       string // 0 = pause, 1 = playing, 2 = recording
	// bitDepth              int
	// sampleRate            int
	// armedChannelCount     int
	// connectedChannelCount int
	// diskTotal             int64
	// diskUsed              int64
	// diskRremainingTime    int64
	// recordingDuration     int64
	// bufferUsagePct        float64
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

func NewTui() *Tui {
	tui := &Tui{
		// channelCount:    0,
		shutdownChannel: make(chan bool, 1),
	}

	return tui
}

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

	tui.tvTransportStatus = custom.NewStatusTextField(headerWidth, "Status", string(runeRecord)+" Recording")
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

// WriteLog should not be used
//
// Deprecated: help me find references to remove
func (tui *Tui) WriteLog(message string) {
	tui.tvLogs.Write([]byte(fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), message)))
}

func (tui *Tui) excecuteLoop() {
	defer tui.app.HandlePanic()

	slog.Info("TUI loop started")

	for {
		if len(tui.shutdownChannel) > 0 || reaper.Reaped() {
			slog.Info("TUI shutting down")
			tui.app.QueueUpdateDraw(func() {})
			break
		}

		tui.app.QueueUpdateDraw(func() {})
		time.Sleep(25 * time.Millisecond)
	}

	fmt.Println("shutting down tui")
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
			tui.meters[i].SetBackgroundColor(meterAlternateBackground)
		}

		tui.metersGrid.AddItem(tui.meters[i], 0, i+1, 1, 1, 0, 0, false)
	}

}

func (tui *Tui) Start() {
	go func() {
		defer tui.app.HandlePanic()

		// Capture user input
		tui.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			// TODO: make sure this works
			// Anything handled here will be executed on the main thread
			switch event.Key() {
			case tcell.KeyEsc:
			case tcell.KeyCtrlC:
				// Exit the application
				tui.app.Stop()
				reaper.Reap()
				return nil
			}

			return event
		})

		if err := tui.app.Run(); err != nil {
			panic(err)
		}

		tui.shutdownChannel <- true
	}()

	go tui.excecuteLoop()
}

func (tui *Tui) IsShutdown() bool {
	return len(tui.shutdownChannel) > 0
}

func (tui *Tui) WaitForShutdown() {
	<-tui.shutdownChannel
}

func format_size(bytes uint64) string {
	suffix := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	// char length = sizeof(suffix) / sizeof(suffix[0])

	i := 0
	bytesFloat := float64(bytes)

	if bytes > 1024 {
		for i = 0; (bytes/1024) > 0 && i < len(suffix); i++ {
			bytesFloat = float64(bytes) / 1024.0
			bytes /= 1024
		}
	}

	return fmt.Sprintf("%.02f %s", bytesFloat, suffix[i])
}

//
// status update functions
//

func (tui *Tui) SetTransportStatus(status int) {
	if status < 0 || status > 3 {
		panic("invalid status value provided: " + string(status))
	}

	var icon rune
	var color tcell.Color
	var transportStatus string

	if status == 0 {
		icon = runePause
		color = theme.Blue
		transportStatus = "Paused"
	} else if status == 1 {
		icon = runePlay
		color = theme.Green
		transportStatus = "Playing"
	} else if status == 2 {
		icon = runeRecord
		color = theme.Red
		transportStatus = "Recording"
	} else if status == 3 {
		icon = runeClock
		color = theme.Yellow
		transportStatus = "Starting Audio Server"
	}

	tui.tvTransportStatus.SetCurrentValue(string(icon) + " " + transportStatus)
	tui.tvTransportStatus.SetColor(color)
}

func (tui *Tui) SetAudioFormat(format string) {
	tui.tvFormat.SetCurrentValue(format)
}

func (tui *Tui) SetSessionSize(size uint64) {
	tui.tvFileSize.SetCurrentValue(format_size(size))
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
