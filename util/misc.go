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
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

func DumpRunes(start int, count int) {
	// 9150
	// 9300
	for i := start; i < start+count; i++ {
		fmt.Printf("%03d %s\n", i, string(rune(i)))
	}
}

func FileExists(path string) bool {
	// if an error occurred or its a directory, we throw up
	if stat, err := os.Stat(path); err != nil || stat.IsDir() {
		return false
	}

	return true
}

func DirectoryExists(testDir string) bool {
	if stat, err := os.Stat(testDir); err != nil || !stat.IsDir() {
		return false
	}

	return true
}

func ResolveHomeDirPath(testPath string) (string, error) {
	if strings.HasPrefix(testPath, "~/") {
		// TODO: make this a shared function
		homeDir, err := os.UserHomeDir()

		if err != nil {
			return "", errors.New("could not find user home dir: " + err.Error())
		}

		return path.Join(homeDir, testPath[2:]), nil
	}

	return testPath, nil
}

func ReadYamlFile(cfg interface{}, fileName string) error {
	filePath := ""

	if path.IsAbs(fileName) {
		filePath = fileName

	} else {
		if strings.HasPrefix(fileName, "~/") {
			testFilePath, err := ResolveHomeDirPath(fileName)
			if err != nil {
				slog.Error(err.Error())
				return err
			}

			if FileExists(testFilePath) {
				filePath = testFilePath
			}

		} else {
			binPath, _ := os.Executable()
			binDir := filepath.Dir(binPath)
			sidecarPath := path.Join(binDir, fileName)

			if FileExists(sidecarPath) {
				filePath = sidecarPath

			} else {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					slog.Error("could not find user home dir: " + err.Error())
					return err
				}

				homeDotConfigPath := path.Join(homeDir, ".config", "fox", fileName)

				if FileExists(homeDotConfigPath) {
					filePath = homeDotConfigPath
				}
			}
		}
	}

	if filePath == "" {
		err := errors.New("no yaml file found")
		slog.Error(err.Error())
		return err
	}

	if !FileExists(filePath) {
		err := errors.New("the specified yaml file does not exist: " + filePath)
		slog.Error(err.Error())
		return err
	}

	slog.Info("Reading yaml from " + filePath)

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(cfg)
	if err != nil {
		return err
	}

	return nil
}

func TraceLog(message string, args ...any) {
	slog.Log(context.Background(), slog.Level(-10), message, args...)
}
