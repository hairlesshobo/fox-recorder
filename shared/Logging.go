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
package shared

import (
	"fmt"
	"os"
	"strings"
	"time"
)

var stockStderr *os.File
var stockStdout *os.File
var logSinks = make([]LogHandler, 0)

type LogHandler func(LogLevel, string)
type LogLevel int8

const (
	ERROR LogLevel = iota
	WARN
	INFO
	DEBUG
)

func (s LogLevel) String() string {
	switch s {
	case ERROR:
		return "Error"
	case WARN:
		return "Warning"
	case INFO:
		return "Info"
	case DEBUG:
		return "Debug"
	}
	return "unknown"
}

//------------------------------------------------------------------
// public functions
//------------------------------------------------------------------

func HijackLogging() {
	stockStdout = os.Stdout
	stockStderr = os.Stderr

	cancelChan := make(chan bool, 1)
	defer close(cancelChan)

	stdout_r, stdout_w, err := os.Pipe()
	if err != nil {
		fmt.Fprintln(stockStderr, err)
	}
	go logProcessor(stdout_r, INFO, cancelChan)

	// stderr_r, stderr_w, err := os.Pipe()
	// if err != nil {
	// 	fmt.Fprintln(stockStderr, err)
	// }
	// go logProcessor(stderr_r, ERROR, cancelChan)

	// TODO: should be routing all output to slog
	os.Stdout = stdout_w
	// os.Stderr = stderr_w
}

func EnableStderrLogging() {
	AddLogSink(stderrLogger)
}

func AddLogSink(fn LogHandler) {
	logSinks = append(logSinks, fn)
}

//------------------------------------------------------------------
// private functions
//------------------------------------------------------------------

func stderrLogger(level LogLevel, message string) {
	dtm := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(stockStdout, "[%s] [%s] %s\n", dtm, level.String(), message)
}

func logProcessor(pipe *os.File, level LogLevel, cancelChan chan bool) {
	buffer := make([]byte, 1024)

out:
	for {
		select {
		case doCancel := <-cancelChan:
			if doCancel {
				break out
			}

			n, err := pipe.Read(buffer)
			if err != nil {
				fmt.Fprintf(stockStderr, "Error reading from pipe: %s\n", err)
			}

			message := strings.TrimSuffix(string(buffer[:n]), "\n")

			for _, logger := range logSinks {
				logger(level, message)
			}
		}

		// data, err := ioutil.ReadFile(filePath)
		// if err != nil {
		// 	fmt.Printf("Error reading file %s: %s\n", filePath, err)
		// 	return
		// }

	}
}
