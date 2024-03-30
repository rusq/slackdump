package dirproc

import (
	"fmt"
	"sync"

	"github.com/rusq/slackdump/v3/internal/chunk"
)

type tracker struct {
	dir *chunk.Directory

	mu    sync.Mutex                   // guards map operations
	files map[chunk.FileID]*entityproc // files holds open files along with their processors
}

func newTracker(cd *chunk.Directory) *tracker {
	return &tracker{
		dir:   cd,
		files: make(map[chunk.FileID]*entityproc),
	}
}

// ensure ensures that the channel file is open and the recorder is
// initialized.
func (t *tracker) create(id chunk.FileID) error {
	if _, ok := t.files[id]; ok {
		// already exists
		return nil
	}
	bp, err := newBaseProc(t.dir, id)
	if err != nil {
		return err
	}
	t.files[id] = &entityproc{
		baseproc: bp,
		refs:     1, // one for the channel
	}
	return nil
}

func (t *tracker) destroy(id chunk.FileID) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.files, id)
}

func (t *tracker) recorder(id chunk.FileID) (*entityproc, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	f, ok := t.files[id]
	if ok {
		return f, nil
	}
	if err := t.create(id); err != nil {
		return nil, err
	}
	return t.files[id], nil
}

func (t *tracker) CloseAll() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	for id, f := range t.files {
		if err := f.Close(); err != nil {
			return fmt.Errorf("error closing %s: %w", id, err)
		}
		delete(t.files, id)
	}
	return nil

}

// RefCount returns the reference count for the given file.
func (t *tracker) RefCount(id chunk.FileID) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	if f, ok := t.files[id]; ok {
		return f.RefCount()
	}
	return 0
}

// entityproc is a processor for a single entity, which can be a thread or
// a channel.
type entityproc struct {
	*baseproc
	// refs is the number of refs are expected to be processed for
	// the given channel.  We keep track of the number of refs, to ensure
	// that we don't close the file until all refs are processed.
	// The channel file can be closed when the number of refs is zero.
	refs int
	mu   sync.Mutex // guards refcnt
}

func (ep *entityproc) AddN(n int) int {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	ep.refs += n
	return ep.refs
}

func (ep *entityproc) Add() int {
	return ep.AddN(1)
}

func (ep *entityproc) DecN(n int) int {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	ep.refs -= n
	return ep.refs
}

func (ep *entityproc) Dec() int {
	return ep.DecN(1)
}

func (ep *entityproc) RefCount() int {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	return ep.refs
}
