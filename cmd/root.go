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
	"math"
	"math/rand/v2"
	"os"
	"time"

	"fox-audio/display"
	"fox-audio/model"
	"fox-audio/shared"

	"github.com/hairlesshobo/go-jack"

	"github.com/spf13/cobra"
)

const jackClientName = "fox"

type displayObj struct {
	tui *display.Tui
}

var (
	argSimulate             bool
	argSimulateChannelCount int
	argSimulateFreezeMeters bool

	displayHandle displayObj
	channels      = 1
	portsIn       []*jack.Port

	// importArgIndividual bool
	// importArgDryRun     bool
	// importArgServer     string
	// importArgDump       bool

	rootCmd = &cobra.Command{
		Use:   "record",
		Short: "Start a recording session",

		Run: func(cmd *cobra.Command, args []string) {
			// config := cmd.Context().Value(model.ImportConfigContext).(model.ImporterConfig)

			run(argSimulate, argSimulateFreezeMeters, argSimulateChannelCount)
		},
	}
)

func init() {
	// ui test commands
	rootCmd.Flags().BoolVarP(&argSimulate, "simulate", "", false, "Freeze the meters (don't randomly set level)")
	rootCmd.Flags().BoolVarP(&argSimulateFreezeMeters, "simulate-freeze-meters", "", false, "Freeze the meters (don't randomly set level)")
	rootCmd.Flags().IntVarP(&argSimulateChannelCount, "simulate-channel-count", "", 32, "Mumber of channels to simulate in UI test")

	// importCmd.Flags().BoolVarP(&importArgIndividual, "individual", "i", false, "Run a single import without connecting to the running server")
	// importCmd.Flags().BoolVarP(&importArgDryRun, "dry_run", "n", false, "Perform a dry-run import (don't copy anything)")
	// importCmd.Flags().BoolVarP(&importArgDump, "dump", "d", false, "If set, dump the list of scanned files to json and exit (for debugging only)")
	// importCmd.Flags().StringVarP(&importArgServer, "server", "s", "localhost:7273", "<host>:<port> -- If specified, connect to the specified server instance to queue an import")
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// ctx := context.WithValue(context.TODO(), model.ImportConfigContext, config)
	// err := rootCmd.ExecuteContext(ctx)

	err := rootCmd.Execute()

	if err != nil {
		os.Exit(1)
	}
}

func ampToDb(amplitude float64) float64 {
	return math.Log10(amplitude) * 20.0
}

func process(nframes uint32) int {
	// loop through the input channels
	levels := make([]*model.SignalLevel, channels)

	for channel, in := range portsIn {

		// get the incoming audio samples
		samplesIn := in.GetBuffer(nframes)

		sigLevel := -150.0

		for frame := range nframes {
			sample := (float64)(samplesIn[frame])
			frameLevel := ampToDb(sample)

			if frameLevel > sigLevel {
				sigLevel = frameLevel
			}
		}

		levels[channel] = &model.SignalLevel{
			Instant: int(sigLevel),
		}
	}

	displayHandle.tui.UpdateSignalLevels(levels)

	return 0
}

func run(simulate bool, simulateFreezeMeters bool, simulateChannelCount int) {
	shared.HijackLogging()
	shared.EnableStderrLogging()

	// fmt.Println("test?")
	// fmt.Println("test?")
	// fmt.Println("test?")

	// time.Sleep(3 * time.Second)

	// return

	if simulate {
		channels = simulateChannelCount
	}

	displayHandle.tui = display.NewTui(channels)

	// this blocks because the tui has to be interactive
	displayHandle.tui.Initalize()
	displayHandle.tui.Start()

	var jackClient *jack.Client
	var jackStatus int
	var jackShutdown chan struct{}

	if !simulate {
		// connect to jack
		jackClient, jackStatus = jack.ClientOpen(jackClientName, jack.NoStartServer)

		if jackStatus != 0 {
			displayHandle.tui.WriteLog(fmt.Sprintf("Status: %s", jack.StrError(jackStatus)))
			return
		}
		defer jackClient.Close()

		// close jack connection on termination
		displayHandle.tui.WriteLog("JACK server connected")

		// register input ports
		for i := 0; i < channels; i++ {
			portIn := jackClient.PortRegister(fmt.Sprintf("in_%d", i), jack.DEFAULT_AUDIO_TYPE, jack.PortIsInput, 0)
			portsIn = append(portsIn, portIn)
		}

		// TODO: get frame time
		// TODO: get sample rate
		// TODO: set error handler

		// set process callback
		if code := jackClient.SetProcessCallback(process); code != 0 {
			displayHandle.tui.WriteLog(fmt.Sprintf("Failed to set process callback: %s", jack.StrError(code)))
			return
		}
		jackShutdown = make(chan struct{})

		// set shutdown handler
		jackClient.OnShutdown(func() {
			displayHandle.tui.WriteLog("JACK connection shutting down")
			close(jackShutdown)
		})

		// activate client
		if code := jackClient.Activate(); code != 0 {
			displayHandle.tui.WriteLog(fmt.Sprintf("Failed to activate client: %s", jack.StrError(code)))
			return
		}

		jackClient.Connect("system:capture_1", fmt.Sprintf("%s:in_0", jackClientName))
	}

	displayHandle.tui.WriteLog("Input ports connected")

	// TODO: connect port(s)

	// this blocks until the jack connection shuts down
	if !simulate {
		<-jackShutdown
	} else {
		startSimulation(simulateFreezeMeters)
	}
	displayHandle.tui.WaitForShutdown()
}

func DumpRunes(start int, count int) {
	// 9150
	// 9300
	for i := start; i < start+count; i++ {
		fmt.Printf("%03d %s\n", i, string(rune(i)))
	}
}

func startSimulation(freezeMeters bool) {
	go func() {
		t := time.NewTicker(150 * time.Millisecond)
		levels := make([]*model.SignalLevel, channels)

		displayHandle.tui.SetTransportStatus(2)
		displayHandle.tui.SetAudioFormat("24 bit / 48k WAV")

		size := uint64(0)

		for range t.C {
			// TODO: have the random meters fall off more gradually to seem more realistic
			size += uint64(rand.IntN(5) * 1024 * 32)

			if displayHandle.tui.IsShutdown() {
				break
			}

			for channel := range channels {
				newLevel := (rand.IntN(70) + 0) * (-1)

				levels[channel] = &model.SignalLevel{
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
}
