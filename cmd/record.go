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
	"time"

	"fox-audio/display"
	"fox-audio/shared"
	"github.com/hairlesshobo/go-jack"

	"github.com/spf13/cobra"
)

const jackClientName = "FoxRecorder"

type displayObj struct {
	tui *display.Tui
}

var displayHandle displayObj

var channels = 1

var portsIn []*jack.Port

var (
	// importArgIndividual bool
	// importArgDryRun     bool
	// importArgServer     string
	// importArgDump       bool

	recordCmd = &cobra.Command{
		Use:   "record",
		Short: "Start a recording session",

		Run: func(cmd *cobra.Command, args []string) {
			// config := cmd.Context().Value(model.ImportConfigContext).(model.ImporterConfig)

		},
	}
)

func init() {
	// importCmd.Flags().BoolVarP(&importArgIndividual, "individual", "i", false, "Run a single import without connecting to the running server")
	// importCmd.Flags().BoolVarP(&importArgDryRun, "dry_run", "n", false, "Perform a dry-run import (don't copy anything)")
	// importCmd.Flags().BoolVarP(&importArgDump, "dump", "d", false, "If set, dump the list of scanned files to json and exit (for debugging only)")
	// importCmd.Flags().StringVarP(&importArgServer, "server", "s", "localhost:7273", "<host>:<port> -- If specified, connect to the specified server instance to queue an import")

	rootCmd.AddCommand(recordCmd)
}

func ampToDb(amplitude float64) float64 {
	return math.Log10(amplitude) * 20.0
}

func process(nframes uint32) int {
	// loop through the input channels
	levels := make([]*shared.SignalLevel, channels)

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

		levels[channel] = &shared.SignalLevel{
			Instant: int(sigLevel),
		}
	}

	displayHandle.tui.UpdateSignalLevels(levels)

	return 0
}

func Main() {
	shared.HijackLogging()
	shared.EnableStderrLogging()

	fmt.Println("test?")
	fmt.Println("test?")
	fmt.Println("test?")

	time.Sleep(3 * time.Second)

	return

	displayHandle.tui = display.NewTui(channels)

	// this blocks because the tui has to be interactive
	ready := make(chan bool)
	go displayHandle.tui.Initalize(ready)

	<-ready
	displayHandle.tui.Start()

	// connect to jack
	client, status := jack.ClientOpen(jackClientName, jack.NoStartServer)

	if status != 0 {
		displayHandle.tui.WriteLog(fmt.Sprintf("Status: %s", jack.StrError(status)))
		return
	}
	// close jack connection on termination
	defer client.Close()
	displayHandle.tui.WriteLog("JACK server connected")

	// register input ports
	for i := 0; i < channels; i++ {
		portIn := client.PortRegister(fmt.Sprintf("in_%d", i), jack.DEFAULT_AUDIO_TYPE, jack.PortIsInput, 0)
		portsIn = append(portsIn, portIn)
	}

	// TODO: get frame time
	// TODO: get sample rate
	// TODO: set error handler

	// set process callback
	if code := client.SetProcessCallback(process); code != 0 {
		displayHandle.tui.WriteLog(fmt.Sprintf("Failed to set process callback: %s", jack.StrError(code)))
		return
	}
	shutdown := make(chan struct{})

	// set shutdown handler
	client.OnShutdown(func() {
		displayHandle.tui.WriteLog("JACK connection shutting down")
		close(shutdown)
	})

	// activate client
	if code := client.Activate(); code != 0 {
		displayHandle.tui.WriteLog(fmt.Sprintf("Failed to activate client: %s", jack.StrError(code)))
		return
	}

	client.Connect("system:capture_1", fmt.Sprintf("%s:in_0", jackClientName))

	displayHandle.tui.WriteLog("Input ports connected")

	// TODO: connect port(s)

	// this blocks until the jack connection shuts down
	<-shutdown
}
