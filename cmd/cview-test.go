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
package cmd

import (
	"fmt"
	"math/rand/v2"

	// "strings"
	"time"

	"fox-audio/custom"
	"fox-audio/display"
	"fox-audio/shared"

	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"

	"github.com/spf13/cobra"
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

var (
	uiTestChannelCount int
	// importArgIndividual bool
	// importArgDryRun     bool
	// importArgServer     string
	// importArgDump       bool

	uiTestCmd = &cobra.Command{
		Use:   "ui-test",
		Short: "Test the terminal user interface (for development)",

		Run: func(cmd *cobra.Command, args []string) {
			// config := cmd.Context().Value(model.ImportConfigContext).(model.ImporterConfig)

			testUi(uiTestChannelCount)
		},
	}
)

func init() {
	// importCmd.Flags().BoolVarP(&importArgIndividual, "individual", "i", false, "Run a single import without connecting to the running server")
	// importCmd.Flags().BoolVarP(&importArgDryRun, "dry_run", "n", false, "Perform a dry-run import (don't copy anything)")
	// importCmd.Flags().BoolVarP(&importArgDump, "dump", "d", false, "If set, dump the list of scanned files to json and exit (for debugging only)")
	// importCmd.Flags().StringVarP(&importArgServer, "server", "s", "localhost:7273", "<host>:<port> -- If specified, connect to the specified server instance to queue an import")
	uiTestCmd.Flags().IntVarP(&uiTestChannelCount, "channel_count", "c", 32, "Mumber of channels to simulate in UI test")

	rootCmd.AddCommand(uiTestCmd)
}

func testUi(channels int) {
	displayHandle.tui = display.NewTui(channels)

	// this blocks because the tui has to be interactive
	ready := make(chan bool)
	go displayHandle.tui.Initalize(ready)

	<-ready
	displayHandle.tui.Start()

	displayHandle.tui.WriteLog("JACK server connected")

	shutdown := make(chan struct{})

	displayHandle.tui.WriteLog("Input ports connected")

	go func() {
		t := time.NewTicker(100 * time.Millisecond)
		levels := make([]*shared.SignalLevel, channels)

		for range t.C {
			for channel := range channels {
				newLevel := rand.IntN(24) * (-1)

				levels[channel] = &shared.SignalLevel{
					Instant: int(newLevel),
				}
			}

			// Queue draw
			displayHandle.tui.UpdateSignalLevels(levels)
		}
	}()

	// this blocks until the jack connection shuts down
	<-shutdown
}

func TestCviewOld() {
	app := cview.NewApplication()
	defer app.HandlePanic()

	meterRowHeight := len(meterSteps) + 3

	grid := cview.NewGrid()
	grid.SetPadding(0, 0, 0, 0)
	grid.SetColumns(-1)
	grid.SetRows(10, meterRowHeight, -1)
	grid.SetBackgroundColor(cview.Styles.PrimitiveBackgroundColor)

	statusView := cview.NewTextView()
	statusView.SetBorder(true)
	statusView.SetPadding(0, 0, 0, 0)
	statusView.Write([]byte("Status    : Recording\n"))
	statusView.Write([]byte("Elapsed   : 00:00:00.000\n"))
	grid.AddItem(statusView, 0, 0, 1, 1, 0, 0, false)

	levelColumns := make([]int, channels+2)
	levelColumns[0] = 5
	for i := range channels {
		levelColumns[i+1] = meterWidth
	}
	levelColumns[channels+1] = -1

	levelsGrid := cview.NewGrid()
	levelsGrid.SetBorder(false)
	levelsGrid.SetBorders(false)
	levelsGrid.SetPadding(0, 0, 0, 0)
	levelsGrid.SetColumns(levelColumns...)
	// levelsGrid.SetRows(2, 0, 1)
	levelsGrid.SetRows(0)
	grid.AddItem(levelsGrid, 1, 0, 1, 1, 0, 0, false)

	logsView := cview.NewTextView()
	logsView.SetPadding(0, 0, 0, 0)
	logsView.SetText("this is the logs view\n")
	grid.AddItem(logsView, 2, 0, 1, 1, 0, 0, false)

	meters := make([]*custom.LevelMeter, channels)

	meterStepLabel := cview.NewTextView()
	meterStepLabel.SetPadding(0, 0, 0, 0)

	// meterStepLabels := make([]string, len(meterSteps))
	meterStepLabel.Write([]byte(fmt.Sprintln()))
	for step := 0; step < len(meterSteps); step++ {
		meterStepLabel.Write([]byte(fmt.Sprintf("%3v\n", fmt.Sprintf("%d", meterSteps[step]))))
		// meterStepLabels = append(meterStepLabels, fmt.Sprintf("%3v", fmt.Sprintf("%d", meterSteps[step])))
	}
	levelsGrid.AddItem(meterStepLabel, 0, 0, 1, 1, 0, 0, false)

	for i := range channels {
		meters[i] = custom.NewLevelMeter(meterSteps, levelColors)
		meters[i].SetBorder(false)
		meters[i].SetPadding(0, 0, 1, 1)
		meters[i].SetLevel(-99)
		meters[i].SetChannelNumber(fmt.Sprintf("%d", i+1))
		if i%2 == 1 {
			meters[i].SetBackgroundColor(tcell.Color233)
		}

		levelsGrid.AddItem(meters[i], 0, i+1, 1, 1, 0, 0, false)
	}

	go func() {
		t := time.NewTicker(100 * time.Millisecond)
		for range t.C {
			for i := range channels {
				newLevel := rand.IntN(24) * (-1)
				meters[i].SetLevel(newLevel)
				if newLevel > meters[i].GetLongTermMaxLevel() {
					meters[i].SetLongTermMaxLevel(newLevel)
				}
			}

			// Queue draw
			app.QueueUpdateDraw(func() {})
		}
	}()

	app.SetRoot(grid, true)
	if err := app.Run(); err != nil {
		panic(err)
	}
}
