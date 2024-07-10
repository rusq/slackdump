// Package dirproc is a processor that writes the data into gzipped files in a
// directory.  Each conversation is output to a separate gzipped JSONL file.
// If a thread is given, the filename will have the thread ID in it.
package dirproc

import (
	"errors"
	"io"
	"sync/atomic"

	"github.com/rusq/slackdump/v3/internal/chunk"
)

// dirproc exposes recording functionality for processor, and handles chunk
// file creation.
type dirproc struct {
	wc     io.WriteCloser
	closed atomic.Bool
	*chunk.Recorder
}

// newDirProc initialises the new base processor which wraps the file
// recorder.  It creates a new chunk file in a directory dir which must exist.
func newDirProc(cd *chunk.Directory, name chunk.FileID) (*dirproc, error) {
	wc, err := cd.Create(name)
	if err != nil {
		return nil, err
	}

	r := chunk.NewRecorder(wc)
	return &dirproc{
		wc:       wc,
		Recorder: r,
	}, nil
}

// Close closes the processor and the underlying chunk file.
func (p *dirproc) Close() error {
	if p.closed.Load() {
		return nil
	}
	var errs error
	if err := p.Recorder.Close(); err != nil {
		errors.Join(errs, err)
	}
	p.closed.Store(true)
	if err := p.wc.Close(); err != nil {
		errors.Join(errs, err)
	}
	return errs
}
