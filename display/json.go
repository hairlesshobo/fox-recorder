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
package display

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"fox-audio/model"
)

//
// types
//

type JsonUI struct {
	shutdownChannel chan bool

	output *os.File

	// sessionName           string
	// jackServerStatus      int    // 0 = not running, 1 = running, 2 = running with warnings, 3 = terminated
	// armedChannelCount     int
	// connectedChannelCount int

	// tvLogs            *cview.TextView
	statusTransport Status

	statusDuration    float64
	statusFormat      string
	statusSessionSize uint64
	statusErrorCount  int
	statusProfileName string
	statusTakeName    string
	statusDirectory   string

	metricDiskUsedPct        int
	metricBufferUsedPct      int
	metricCycleBufferUsedPct int
	metricAudioLoadPct       int
	metricDiskLoadPct        int

	signalLevels []model.SignalLevel
	outputFiles  []model.UiOutputFile
}

//
// constructor
//

func NewJsonUI(output *os.File) *JsonUI {

	jsonUi := &JsonUI{
		shutdownChannel: make(chan bool, 1),

		output: output,

		statusTransport: StatusStarting,

		statusDuration:    0.0,
		statusFormat:      "",
		statusSessionSize: 0,
		statusErrorCount:  0,
		statusProfileName: "",
		statusTakeName:    "",
		statusDirectory:   "",

		metricDiskUsedPct:        0,
		metricBufferUsedPct:      0,
		metricCycleBufferUsedPct: 0,
		metricAudioLoadPct:       0,
		metricDiskLoadPct:        0,

		signalLevels: make([]model.SignalLevel, 0),
		outputFiles:  make([]model.UiOutputFile, 0),
		// elementLevelMeters: make([]*custom.LevelMeter, 0),
		// elementOutputFiles: make([]*custom.OutputFileField, 0),
	}

	return jsonUi
}

func (j *JsonUI) Initalize() {
	// nothing to do here
}

func (j *JsonUI) Start() {
	go j.excecuteLoop()
}

func (j *JsonUI) excecuteLoop() {
	slog.Debug("JSON loop started")

	for {
		if len(j.shutdownChannel) > 0 {
			slog.Info("JSON UI shutting down")
			break
		}

		j.printJson(j.getStatus())
		j.printJson(j.getLevels())
		j.printJson(j.getOutputFiles())

		time.Sleep(1 * time.Second)
	}

	fmt.Println("shutting down tui")
}

func (j *JsonUI) Shutdown() {
	slog.Debug("Shutting down JSON UI")
	j.shutdownChannel <- true

	slog.Debug("Waiting for JSON UI to shut down")
	j.WaitForShutdown()
}

func (j *JsonUI) IsShutdown() bool {
	return len(j.shutdownChannel) > 0
}

func (j *JsonUI) WaitForShutdown() {
	<-j.shutdownChannel
}

func (j *JsonUI) SetTransportStatus(status Status) {
	j.statusTransport = status
}

func (j *JsonUI) SetDuration(duration float64) {
	j.statusDuration = duration
}

func (j *JsonUI) SetAudioFormat(format string) {
	j.statusFormat = format
}

func (j *JsonUI) SetProfileName(value string) {
	j.statusProfileName = value
}

func (j *JsonUI) SetTakeName(value string) {
	j.statusTakeName = value
}

func (j *JsonUI) SetDirectory(value string) {
	j.statusDirectory = value
}

func (j *JsonUI) SetSessionSize(size uint64) {
	j.statusSessionSize = size
}

func (j *JsonUI) IncrementErrorCount() {
	j.statusErrorCount += 1
}

func (j *JsonUI) UpdateSignalLevels(levels []model.SignalLevel) {
	copy(levels, j.signalLevels)
}

func (j *JsonUI) SetChannelArmStatus(channel int, armed bool) {
	// nothing to do here
}

func (j *JsonUI) SetOutputFiles(outputFiles []model.UiOutputFile) {
	j.outputFiles = make([]model.UiOutputFile, len(outputFiles))

	for i, file := range outputFiles {
		j.outputFiles[i] = file
	}
}

func (j *JsonUI) UpdateOutputFileSizes(sizes []uint64) {
	for i, size := range sizes {
		j.outputFiles[i].Size = size
	}
}

func (j *JsonUI) SetChannelCount(channelCount int) {
	j.signalLevels = make([]model.SignalLevel, channelCount)
}

func (j *JsonUI) WriteLevelLog(level slog.Level, message string) {
	logObj := JsonLog{
		MessageType: "log",

		Date:    time.Now().Format(time.RFC3339),
		Level:   level.String(),
		Message: message,
	}

	j.printJson(logObj)
}

func (j *JsonUI) SetAudioLoad(percent int) {
	j.metricAudioLoadPct = percent
}

func (j *JsonUI) SetDiskUsage(percent int) {
	j.metricDiskUsedPct = percent
}

func (j *JsonUI) SetBufferUtilization(percent int) {
	j.metricBufferUsedPct = percent
}

func (j *JsonUI) SetDiskLoad(percent int) {
	j.metricDiskLoadPct = percent
}

func (j *JsonUI) SetCycleBuffer(percent int) {
	j.metricCycleBufferUsedPct = percent
}

//
// private functions
//

func (j *JsonUI) printJson(v any) {
	jsonBytes, err := json.Marshal(v)

	if err != nil {
		slog.Error("Error marshalling to JSON: " + err.Error())
	}

	fmt.Fprintln(j.output, string(jsonBytes))
}

func (j *JsonUI) getStatus() *JsonStatus {
	jsonStatus := &JsonStatus{
		MessageType: "status",

		Status: statusNames[j.statusTransport],

		Duration:    j.statusDuration,
		Format:      j.statusFormat,
		SessionSize: j.statusSessionSize,
		ErrorCount:  j.statusErrorCount,
		ProfileName: j.statusProfileName,
		TakeName:    j.statusTakeName,
		Directory:   j.statusDirectory,

		DiskUsedPct:        j.metricDiskUsedPct,
		BufferUsedPct:      j.metricBufferUsedPct,
		CycleBufferUsedPct: j.metricCycleBufferUsedPct,
		AudioLoadPct:       j.metricAudioLoadPct,
		DiskLoadPct:        j.metricDiskLoadPct,
	}

	return jsonStatus
}

func (j *JsonUI) getLevels() *JsonLevels {
	jsonLevels := &JsonLevels{
		MessageType: "levels",

		Ports: make([]JsonLevelPort, len(j.signalLevels)),
	}

	for i, level := range j.signalLevels {
		jsonLevels.Ports[i].Name = fmt.Sprintf("%d", i+1)
		jsonLevels.Ports[i].Level = level.Instant
	}

	return jsonLevels
}

func (j *JsonUI) getOutputFiles() *JsonOutputFiles {
	outputFiles := &JsonOutputFiles{
		MessageType: "files",

		Files: make([]JsonOutputFile, len(j.outputFiles)),
	}

	for i, file := range j.outputFiles {
		outputFiles.Files[i].Name = file.Name
		outputFiles.Files[i].Ports = file.Ports
		outputFiles.Files[i].Size = file.Size
	}

	return outputFiles
}
