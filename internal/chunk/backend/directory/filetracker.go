package directory

import (
	"fmt"
	"sync"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/primitive"
)

// filetracker keeps track of the files and their processors.
type filetracker struct {
	dir *chunk.Directory

	mu    sync.RWMutex                 // guards map operations
	files map[chunk.FileID]*entityproc // files holds open files along with their processors
}

// entityproc is a processor for a single entity, which can be a thread or
// a channel.
type entityproc struct {
	*dirproc
	// Counter holds the number threads expected to be processed for the given
	// channel.  We keep track of the number of threads, to ensure that we
	// don't close the file until all threads are processed.  The channel file
	// can be closed when the Counter is zero.
	primitive.Counter
}

func newFileTracker(cd *chunk.Directory) *filetracker {
	return &filetracker{
		dir:   cd,
		files: make(map[chunk.FileID]*entityproc),
	}
}

// create ensures that the channel file is open and the recorder is
// initialized.
func (t *filetracker) create(id chunk.FileID) error {
	if _, ok := t.files[id]; ok {
		// already exists
		return nil
	}
	bp, err := newDirProc(t.dir, id)
	if err != nil {
		return err
	}
	ep := &entityproc{
		dirproc: bp,
	}
	ep.Inc() // one for the initial call
	t.files[id] = ep
	return nil
}

// Unregister closes and removes the file from tracking (file remains on the file
// system).
func (t *filetracker) Unregister(id chunk.FileID) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.unregister(id)
}

// unregister is an internal function that closes and removes the file from
// tracking without locking the mutex.
func (t *filetracker) unregister(id chunk.FileID) error {
	r, ok := t.files[id]
	if !ok {
		return nil
	}
	if err := r.Close(); err != nil {
		return err
	}
	delete(t.files, id)
	return nil
}

// Recorder returns the processor for the given file.  If the processor
// doesn't exist, it is created.
func (t *filetracker) Recorder(id chunk.FileID) (datahandler, error) {
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

// CloseAll closes all open files.
func (t *filetracker) CloseAll() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	for id := range t.files {
		if err := t.unregister(id); err != nil {
			return fmt.Errorf("error closing file %s: %w", id, err)
		}
	}
	return nil

}

// RefCount returns the reference count for the given file.
func (t *filetracker) RefCount(id chunk.FileID) int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if f, ok := t.files[id]; ok {
		return f.N()
	}
	return 0
}
