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
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"time"
)

type LogHandler func(LogLevel, string)
type LogLevel int8

const (
	ERROR LogLevel = iota
	WARN
	INFO
	DEBUG
)

var (
	stockStderr *os.File
	stockStdout *os.File
	logSinks    = make([]LogHandler, 0)
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

	stdout_r, stdout_w, err := os.Pipe()
	if err != nil {
		fmt.Fprintln(stockStderr, err)
	}
	go logProcessor(stdout_r, INFO)

	stderr_r, stderr_w, err := os.Pipe()
	if err != nil {
		fmt.Fprintln(stockStderr, err)
	}
	go logProcessor(stderr_r, ERROR)

	os.Stdout = stdout_w
	os.Stderr = stderr_w
}

func EnableSlogLogging() {
	AddLogSink(slogLogger)
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

func slogLogger(level LogLevel, message string) {
	if level == ERROR {
		slog.Error(message)
	} else if level == WARN {
		slog.Warn(message)
	} else if level == INFO {
		slog.Info(message)
	} else if level == DEBUG {
		slog.Debug(message)
	}
}

func logProcessor(pipe *os.File, level LogLevel) {
	scanner := bufio.NewScanner(pipe)

	for scanner.Scan() {
		line := scanner.Text()

		for _, logger := range logSinks {
			logger(level, line)
		}
	}
}
