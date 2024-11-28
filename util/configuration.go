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
package util

import (
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"time"

	"fox-audio/model"
	"fox-audio/reaper"
)

func ReadProfile(profilePath string) *model.Profile {
	if !strings.HasSuffix(profilePath, ".profile") {
		profilePath += ".profile"
	}

	profile := &model.Profile{}

	ReadYamlFile(profile, profilePath)

	prepareOutputDirectory(profile)

	return profile
}

func ReadConfig(args *model.CommandLineArgs) *model.Config {
	outputTypes := make([]string, len(model.OutputTypeMap))

	i := 0
	for key := range model.OutputTypeMap {
		outputTypes[i] = strings.ToLower(key)
		i++
	}

	if !slices.Contains(outputTypes, strings.ToLower(args.OutputType)) {
		slog.Error("Invalid output type specified: " + args.OutputType + ". Valid options: " + strings.Join(outputTypes, ", "))
		os.Exit(1)
	}

	config := &model.Config{
		JackdBinary:                  "",
		VerboseJackServer:            false,
		JackClientName:               "fox",
		ProfileDirectory:             "",
		LogLevel:                     int(slog.LevelInfo),
		OutputType:                   model.OutputTUI,
		HardwarePortConnectionPrefix: "system:capture_", //"multiplier:out",
		SimulationOptions: &model.SimulationOptions{
			EnableSimulation: false,
			FreezeMeters:     false,
			ChannelCount:     32,
		},
	}

	ReadYamlFile(config, args.ConfigFile)

	if config.JackdBinary == "" {
		config.JackdBinary = FindJackdBinary()
	}

	requestedOutputType := model.OutputTypeMap[args.OutputType]
	if requestedOutputType != config.OutputType {
		config.OutputType = requestedOutputType
	}

	if args.Simulate != config.SimulationOptions.EnableSimulation {
		config.SimulationOptions.EnableSimulation = args.Simulate
	}

	if args.SimulateChannelCount != config.SimulationOptions.ChannelCount {
		config.SimulationOptions.ChannelCount = args.SimulateChannelCount
	}

	if args.SimulateFreezeMeters != config.SimulationOptions.FreezeMeters {
		config.SimulationOptions.FreezeMeters = args.SimulateFreezeMeters
	}

	return config
}

func prepareOutputDirectory(profile *model.Profile) {
	var err error
	outputDir, err := ResolveHomeDirPath(time.Now().Format(profile.Output.DirectoryTemplate))
	if err != nil {
		slog.Error("Failed to resolve home user dir: " + err.Error())
		reaper.Reap()
		return
	}

	if !DirectoryExists(outputDir) {
		slog.Info("Creating output directory: " + outputDir)
		os.MkdirAll(outputDir, 0755)
	}

	// set the calculated values in the profile for other parts of the app to use
	profile.Output.Take = getTake(outputDir)
	profile.Output.Directory = outputDir
}

func getTake(outputDir string) string {
	entries, _ := os.ReadDir(outputDir)

	take := byte('A')

out:
	for {
		for _, entry := range entries {
			name := entry.Name()

			// skip directories or non-wav files
			if entry.IsDir() || !strings.HasSuffix(name, ".wav") {
				continue
			}

			if strings.HasPrefix(name, fmt.Sprintf("%s_channel", string(take))) {
				take += 1
				continue out
			}
		}
		break out
	}

	return string(take)
}
