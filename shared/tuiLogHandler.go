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
	"context"
	"log/slog"

	"fox-audio/display"
)

type TuiLogHandler struct {
	level         slog.Level
	tui           *display.Tui
	errorCallback func(string)
}

func NewTuiLogHandler(out *display.Tui, level slog.Level, errorCallback func(string)) *TuiLogHandler {
	h := &TuiLogHandler{
		level:         level,
		tui:           out,
		errorCallback: errorCallback,
	}

	return h
}

func (h *TuiLogHandler) Handle(ctx context.Context, r slog.Record) error {
	h.tui.WriteLevelLog(r.Level, r.Message)

	if r.Level == slog.LevelError {
		h.errorCallback(r.Message)
	}

	return nil
}

func (h *TuiLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *TuiLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *TuiLogHandler) WithGroup(name string) slog.Handler {
	return h
}
