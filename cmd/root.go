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
	"log/slog"
	"math"
	"math/rand/v2"
	"os"
	"syscall"
	"time"

	"fox-audio/audio"
	"fox-audio/display"
	"fox-audio/model"
	"fox-audio/reaper"
	"fox-audio/shared"

	"github.com/spf13/cobra"
)

const jackClientName = "fox"

type displayObj struct {
	tui *display.Tui
}

var (
	// arguments
	argSimulate             bool
	argSimulateChannelCount int
	argSimulateFreezeMeters bool

	displayHandle displayObj
	ports         []*audio.Port
	stats         statistics

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

func jackError(message string) {
	slog.Error("JACK: " + message)
}

func jackInfo(message string) {
	slog.Info("JACK: " + message)
}

func jackShutdown(server *audio.JackServer) {
	slog.Info("JACK connection shutting down")
	server.StopServer()
}

func jackXrun() int {
	slog.Error("xrun")

	return 0
}

type statistics struct {
	lastStartTime      int64
	lastEndTime        int64
	processElapsedChan chan int64
	processIdleChan    chan int64
}

func getChanAverage(inputChan chan int64) float64 {
	sum := 0.0
	count := 0

out:
	for {
		select {
		case value := <-inputChan:
			sum += float64(value)
			count++
		default:
			break out
		}
	}

	return sum / float64(count)
}

func processOnInterval(milliseconds int, process func()) {
	go func() {
		reaper.Register()

		process()

		t := time.NewTicker(time.Duration(milliseconds) * time.Millisecond)

		for range t.C {
			if reaper.Reaped() {
				break
			}

			process()
		}

		reaper.Done()
	}()
}

func initStatistics() {
	stats = statistics{
		processElapsedChan: make(chan int64, 30),
		processIdleChan:    make(chan int64, 30),
	}

	// audio engine load calculations
	processOnInterval(250, func() {
		idleTimeAvg := getChanAverage(stats.processIdleChan)
		processTimeAvg := getChanAverage(stats.processElapsedChan)
		avgAudioLoad := processTimeAvg / idleTimeAvg

		displayHandle.tui.SetAudioLoad(int(avgAudioLoad * 100.0))
		slog.Info(fmt.Sprintf("Idle time: %0.0f us, Process time: %0.0f us, load %0.3f%%", idleTimeAvg, processTimeAvg, avgAudioLoad))
	})

	// disk space utilization
	processOnInterval(1000, func() {
		// TODO: this should point to recording directory
		wd, _ := os.Getwd()

		stat := syscall.Statfs_t{}
		syscall.Statfs(wd, &stat)

		// TODO: make this linux happy
		// TODO: Verify mac accuracy
		diskSize := uint64(stat.Bsize) * (stat.Blocks) // - stat.Bfree - stat.Bavail)
		diskFree := uint64(stat.Bsize) * stat.Bavail
		diskUsed := diskSize - diskFree

		// Available blocks * size per block = available space in bytes
		diskUsedPct := float64(diskUsed) / float64(diskSize)
		displayHandle.tui.SetDiskUsage(int(math.Round(diskUsedPct * 100.0)))

		slog.Info(fmt.Sprintf("Disk total: %d B, Disk Used: %d B, Disk free: %d B, used %0.2f%%", diskSize, diskUsed, diskFree, diskUsedPct))
	})
}

func jackProcess(nframes uint32) int {
	// audio load statistics
	if stats.lastEndTime > 0 {
		if len(stats.processIdleChan) < cap(stats.processIdleChan) {
			stats.processIdleChan <- time.Now().UnixMicro() - stats.lastEndTime
		}
	}

	stats.lastStartTime = time.Now().UnixMicro()

	// loop through the input channels
	levels := make([]*model.SignalLevel, len(ports))

	for portNum, port := range ports {

		// get the incoming audio samples
		samplesIn := port.GetJackPort().GetBuffer(nframes)

		sigLevel := -150.0

		for frame := range nframes {
			sample := (float64)(samplesIn[frame])
			frameLevel := ampToDb(sample)

			if frameLevel > sigLevel {
				sigLevel = frameLevel
			}
		}

		levels[portNum] = &model.SignalLevel{
			Instant: int(sigLevel),
		}
	}

	displayHandle.tui.UpdateSignalLevels(levels)

	// audio load statistics
	stats.lastEndTime = time.Now().UnixMicro()
	if len(stats.processElapsedChan) < cap(stats.processElapsedChan) {
		stats.processElapsedChan <- stats.lastEndTime - stats.lastStartTime

	}
	return 0
}

func run(simulate bool, simulateFreezeMeters bool, simulateChannelCount int) {
	displayHandle.tui = display.NewTui()
	displayHandle.tui.Initalize()
	displayHandle.tui.SetTransportStatus(3)
	displayHandle.tui.Start()

	initStatistics()

	// handler := shared.NewTuiLogHandler(displayHandle.tui, slog.LevelDebug)
	f, err := os.Create("/Users/flip/projects/personal/fox-recorder/fox.log")

	if err != nil {
		panic(err)
	}

	handler := slog.NewTextHandler(f, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	shared.HijackLogging()
	shared.EnableSlogLogging()

	var audioServer *audio.JackServer

	if !simulate {
		audioServer = audio.NewServer(jackClientName, "coreaudio/", 44100, 2048)
		audioServer.SetErrorCallback(jackError)
		audioServer.SetInfoCallback(jackInfo)

		shared.CatchSigint(func() {
			fmt.Println("I don't wanna!")
			audioServer.StopServer()
			os.Exit(0)
		})

		audioServer.StartServer()
		reaper.Callback(audioServer.StopServer)

		displayHandle.tui.SetChannelCount(len(audioServer.GetInputPorts()))
		displayHandle.tui.SetTransportStatus(0)

		audioServer.Connect()
		reaper.Callback(audioServer.Disconnect)

		// only register input ports, for now
		audioServer.RegisterPorts(true, false)
		ports = audioServer.GetInputPorts()

		// // TODO: get frame time
		// // TODO: get sample rate
		// // TODO: set error handler

		// set process callback
		audioServer.SetProcessCallback(jackProcess)

		// set shutdown handler
		audioServer.SetXrunCallback(jackXrun)
		audioServer.SetShutdownCallback(func() { jackShutdown(audioServer) })

		audioServer.ActivateClient()

		audioServer.ConnectPorts(true, false)

		slog.Info("Input ports connected")
	}

	// TODO: connect port(s)

	// this blocks until the jack connection shuts down
	if simulate {
		displayHandle.tui.SetChannelCount(simulateChannelCount)
		startSimulation(simulateFreezeMeters, simulateChannelCount)
	}
	// displayHandle.tui.WaitForShutdown()
	// audioServer.StopServer()
	reaper.Wait()
}

func DumpRunes(start int, count int) {
	// 9150
	// 9300
	for i := start; i < start+count; i++ {
		fmt.Printf("%03d %s\n", i, string(rune(i)))
	}
}

func startSimulation(freezeMeters bool, channelCount int) {
	go func() {
		reaper.Register()
		t := time.NewTicker(150 * time.Millisecond)
		levels := make([]*model.SignalLevel, channelCount)

		displayHandle.tui.SetTransportStatus(2)
		displayHandle.tui.SetAudioFormat("24 bit / 48k WAV")

		size := uint64(0)

		for range t.C {
			// TODO: have the random meters fall off more gradually to seem more realistic
			size += uint64(rand.IntN(5) * 1024 * 32)

			if reaper.Reaped() {
				break
			}

			for channel := range channelCount {
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

		reaper.Done()
	}()
}
