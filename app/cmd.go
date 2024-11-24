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
package app

import (
	"log/slog"
	"os"

	"fox-audio/util"

	"github.com/spf13/cobra"
)

var (
	// arguments
	argSimulate             bool
	argSimulateChannelCount int
	argSimulateFreezeMeters bool
	argProfileName          string

	rootCmd = &cobra.Command{
		Use:   "record",
		Short: "Start a recording session",

		Run: func(cmd *cobra.Command, args []string) {
			// config := cmd.Context().Value(model.ImportConfigContext).(model.ImporterConfig)
			if argProfileName == "" {
				slog.Error("Profile not specified but is REQUIRED. See fox --help for more info")
				os.Exit(1)
			}

			profile := util.ReadProfile(argProfileName)

			runEngine(profile, argSimulate, argSimulateFreezeMeters, argSimulateChannelCount)
		},
	}
)

func init() {
	// ui test commands
	rootCmd.Flags().BoolVar(&argSimulate, "simulate", false, "Freeze the meters (don't randomly set level)")
	rootCmd.Flags().BoolVar(&argSimulateFreezeMeters, "simulate-freeze-meters", false, "Freeze the meters (don't randomly set level)")
	rootCmd.Flags().IntVar(&argSimulateChannelCount, "simulate-channel-count", 32, "Mumber of channels to simulate in UI test")

	rootCmd.Flags().StringVarP(&argProfileName, "profile", "p", "", "Name or path of the profile to load, REQUIRED")
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
