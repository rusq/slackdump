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
