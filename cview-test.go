package main

import (
	"fmt"
	"math/rand/v2"
	// "strings"
	"time"

	"github.com/hairlesshobo/fox-recorder/custom"

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

func testCview() {
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
