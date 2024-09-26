// The main package for FoxRecorder
package main

import (
	"fmt"
	"math"

	"github.com/hairlesshobo/fox-recorder/display"
	"github.com/xthexder/go-jack"
)

const jackClientName = "FoxRecorder"

type channel struct {
	number    int
	inputName string
}

type displayObj struct {
	tui *display.Tui
}

var displayHandle displayObj

var channels = 1

var portsIn []*jack.Port

func ampToDb(amplitude float64) float64 {
	return math.Log10(amplitude) * 20.0
}

func process(nframes uint32) int {
	// loop through the input channels
	levels := make([]*display.SignalLevel, channels)

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

		levels[channel] = &display.SignalLevel{
			Instant: int(sigLevel),
		}
	}

	displayHandle.tui.UpdateSignalLevels(levels)

	return 0
}

func main() {
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
