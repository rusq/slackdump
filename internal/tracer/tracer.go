// Package tracer is simple convenience wrapper around writing trace to a file.
package tracer

import (
	"fmt"
	"os"
	"runtime/trace"
)

type Info struct {
	Filename string

	tf *os.File
}

func New(tracefile string) *Info {
	return &Info{Filename: tracefile}
}

// Start starts the tracing
func (t *Info) Start() error {
	var err error
	t.tf, err = os.Create(t.Filename)
	if err != nil {
		return fmt.Errorf("failed to create trace output file: %w", err)
	}
	if err := trace.Start(t.tf); err != nil {
		return fmt.Errorf("failed to start trace: %w", err)
	}
	return nil
}

func (t *Info) End() error {
	if t.tf == nil {
		return nil
	}
	if trace.IsEnabled() {
		trace.Stop()
	}
	if err := t.tf.Close(); err != nil {
		return err
	}
	t.tf = nil
	return nil
}
