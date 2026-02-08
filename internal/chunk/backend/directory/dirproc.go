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
// Package directory is a processor that writes the data into gzipped files in a
// directory.  Each conversation is output to a separate gzipped JSONL file.
// If a thread is given, the filename will have the thread ID in it.
package directory

import (
	"errors"
	"io"
	"sync/atomic"

	"github.com/rusq/slackdump/v4/internal/chunk"
)

// dirproc exposes recording functionality for processor, and handles chunk
// file creation.
type dirproc struct {
	wc     io.WriteCloser
	closed atomic.Bool
	*chunk.Recorder
}

// newDirProc initialises the new directory processor which wraps the file
// recorder.  It creates a new chunk file in a directory cd. Directory cd
// must exist.
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
	if !p.closed.CompareAndSwap(false, true) {
		return nil
	}
	var errs error
	if err := p.Recorder.Close(); err != nil {
		errs = errors.Join(errs, err)
	}
	if err := p.wc.Close(); err != nil {
		errs = errors.Join(errs, err)
	}
	return errs
}
