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
	"math/rand/v2"
	"time"

	"fox-audio/display"
	"fox-audio/shared"

	"github.com/spf13/cobra"
)

var (
	uiTestChannelCount int
	uiTestFreezeMeters bool

	uiTestCmd = &cobra.Command{
		Use:   "ui-test",
		Short: "Test the terminal user interface (for development)",

		Run: func(cmd *cobra.Command, args []string) {
			// config := cmd.Context().Value(model.ImportConfigContext).(model.ImporterConfig)

			testUi(uiTestChannelCount, uiTestFreezeMeters)
		},
	}
)

func init() {
	uiTestCmd.Flags().BoolVarP(&uiTestFreezeMeters, "freeze_meters", "f", false, "Freeze the meters (don't randomly set level)")
	uiTestCmd.Flags().IntVarP(&uiTestChannelCount, "channel_count", "c", 32, "Mumber of channels to simulate in UI test")

	rootCmd.AddCommand(uiTestCmd)
}

func testUi(channels int, freezeMeters bool) {

	// for i := 9150; i < 9300; i++ {
	// 	fmt.Printf("%03d %s\n", i, string(rune(i)))
	// }

	// fmt.Println('⏸')

	// return
	displayHandle.tui = display.NewTui(channels)

	// this blocks because the tui has to be interactive
	displayHandle.tui.Initalize()
	displayHandle.tui.Start()

	displayHandle.tui.WriteLog("JACK server connected")

	displayHandle.tui.WriteLog("Input ports connected")

	go func() {
		t := time.NewTicker(100 * time.Millisecond)
		levels := make([]*shared.SignalLevel, channels)

		displayHandle.tui.SetTransportStatus(2)
		displayHandle.tui.SetAudioFormat("24 bit / 48k WAV")

		size := uint64(0)
	out:
		for range t.C {
			size += uint64(rand.IntN(5) * 1024 * 32)

			if displayHandle.tui.IsShutdown() {
				break out
			}

			for channel := range channels {
				newLevel := rand.IntN(24) * (-1)

				levels[channel] = &shared.SignalLevel{
					Instant: int(newLevel),
				}
			}

			// Queue draw
			displayHandle.tui.UpdateSignalLevels(levels)
			displayHandle.tui.SetSessionSize(size)

			if freezeMeters {
				break
			}
		}
	}()

	// this blocks until the jack connection shuts down
	displayHandle.tui.WaitForShutdown()
}
