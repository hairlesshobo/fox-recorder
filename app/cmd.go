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
	"os"

	"github.com/spf13/cobra"
)

var (
	// arguments
	argSimulate             bool
	argSimulateChannelCount int
	argSimulateFreezeMeters bool

	rootCmd = &cobra.Command{
		Use:   "record",
		Short: "Start a recording session",

		Run: func(cmd *cobra.Command, args []string) {
			// config := cmd.Context().Value(model.ImportConfigContext).(model.ImporterConfig)

			runEngine(argSimulate, argSimulateFreezeMeters, argSimulateChannelCount)
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
