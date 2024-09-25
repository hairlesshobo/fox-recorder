// The main package for FoxRecorder
package main

import (
	"fmt"
	"math"

	"github.com/xthexder/go-jack"
)

const jackClientName = "FoxRecorder"

type channel struct {
	number    int
	inputName string
}

var channels = 32

var portsIn []*jack.Port

func ampToDb(amplitude float64) float64 {
	return math.Log10(amplitude) * 20.0
}

func process(nframes uint32) int {
	// loop through the input channels
	for _, in := range portsIn {

		// get the incoming audio samples
		samplesIn := in.GetBuffer(nframes)

		sigLevel := -128.0

		for frame := range nframes {
			sample := (float64)(samplesIn[frame])
			frameLevel := ampToDb(sample)

			if frameLevel > sigLevel {
				sigLevel = frameLevel
			}
		}

		fmt.Printf("level: %1.0f\n", sigLevel)
	}

	return 0
}

func main() {
	testCview()

	return
	// connect to jack
	client, status := jack.ClientOpen(jackClientName, jack.NoStartServer)

	if status != 0 {
		fmt.Println("Status:", jack.StrError(status))
		return
	}
	// close jack connection on termination
	defer client.Close()

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
		fmt.Println("Failed to set process callback:", jack.StrError(code))
		return
	}
	shutdown := make(chan struct{})

	// set shutdown handler
	client.OnShutdown(func() {
		fmt.Println("Shutting down")
		close(shutdown)
	})

	// activate client
	if code := client.Activate(); code != 0 {
		fmt.Println("Failed to activate client:", jack.StrError(code))
		return
	}

	client.Connect("system:capture_1", fmt.Sprintf("%s:in_0", jackClientName))

	// TODO: connect port(s)

	fmt.Println(client.GetName())
	<-shutdown
}
