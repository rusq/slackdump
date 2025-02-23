package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/rusq/tracer"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
)

// initLog initialises the logging and returns the context with the Logger. If the
// filename is not empty, the file will be opened, and the logger output will
// be switch to that file. Returns the initialised logger, stop function and
// an error, if any. The stop function must be called in the deferred call, it
// will close the log file, if it is open. If the error is returned the stop
// function is nil.
func initLog(filename string, jsonHandler bool, verbose bool) (*slog.Logger, error) {
	if verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
	opts := &slog.HandlerOptions{
		Level: iftrue(verbose, slog.LevelDebug, slog.LevelInfo),
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
