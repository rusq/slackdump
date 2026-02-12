// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/rusq/tracer"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/internal/primitive"
)

// initLog initialises the logging and returns the context with the Logger. If the
// filename is not empty, the file will be opened, and the logger output will
// be switch to that file. Returns the initialised logger, stop function and
// an error, if any. The stop function must be called in the deferred call, it
// will close the log file, if it is open. If the error is returned the stop
// function is nil.
func initLog(filename string, jsonHandler bool, verbose bool) (*slog.Logger, error) {
	if verbose {
		cfg.SetDebugLevel()
	}
	opts := &slog.HandlerOptions{
		Level: primitive.IfTrue(verbose, slog.LevelDebug, slog.LevelInfo),
	}
	if jsonHandler {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, opts)))
	}
	if filename != "" {
		slog.Debug("log messages will be written to file", "filename", filename)
		lf, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o666)
		if err != nil {
			return slog.Default(), fmt.Errorf("failed to create the log file: %w", err)
		}
		log.SetOutput(lf) // redirect the standard log to the file just in case, panics will be logged there.

		var h slog.Handler = slog.NewTextHandler(lf, opts)
		if jsonHandler {
			h = slog.NewJSONHandler(lf, opts)
		}

		sl := slog.New(h)
		slog.SetDefault(sl)
		base.AtExit(func() {
			if err := lf.Close(); err != nil {
				slog.Error("failed to close the log file", "error", err)
			}
		})
	}

	return slog.Default(), nil
}

// initTrace initialises the tracing.  If the filename is not empty, the file
// will be opened, trace will write to that file.  Returns the stop function
// that must be called in the deferred call.  If the error is returned the stop
// function is nil.
func initTrace(filename string) (stop func()) {
	stop = func() {}
	if filename == "" {
		return
	}

	slog.Info("trace will be written to", "filename", filename)

	trc := tracer.New(filename)
	if err := trc.Start(); err != nil {
		slog.Warn("failed to start the trace", "filename", filename, "error", err)
		return
	}

	stop = func() {
		if err := trc.End(); err != nil {
			slog.Warn("failed to write the trace file", "filename", filename, "error", err)
		}
	}
	return
}
