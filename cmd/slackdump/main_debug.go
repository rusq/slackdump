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
//go:build debug

package main

import (
	"log/slog"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
)

func initDebug() {
	cpuProfStop := initCPUProfile(cfg.CPUProfile)
	base.AtExit(cpuProfStop)
	if cfg.MEMProfile != "" {
		base.AtExit(func() {
			writeMemProfile(cfg.MEMProfile)
		})
	}
}

func initCPUProfile(filename string) (stop func()) {
	stop = func() {}
	if filename == "" {
		return
	}

	slog.Info("cpu profile will be written to", "filename", filename)

	f, err := os.Create(filename)
	if err != nil {
		slog.Warn("could not create CPU profile", "error", err)
		return
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		slog.Warn("could not start CPU profile", "error", err)
	}
	stop = func() {
		pprof.StopCPUProfile()
		if err := f.Close(); err != nil {
			slog.Warn("could not close CPU profile", "error", err)
		}
	}
	return
}

func writeMemProfile(filename string) {
	if filename == "" {
		return
	}

	slog.Info("mem profile will be written to", "filename", filename)

	f, err := os.Create(filename)
	if err != nil {
		slog.Warn("could not create memory profile", "error", err)
		return
	}
	defer f.Close()
	runtime.GC()
	runtime.GC() // get up-to-date statistics
	if err := pprof.Lookup("heap").WriteTo(f, 0); err != nil {
		slog.Warn("could not write memory profile", "error", err)
	}
	if err := f.Close(); err != nil {
		slog.Warn("could not close memory profile", "error", err)
	}
}
