package chunk

import (
	"compress/gzip"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
)

// filemgr manages temporary files and handles for compressed files.
type filemgr struct {
	tmpdir  string               // temporary storage directory
	once    *sync.Once           // ensures that the temporary directory is created only once
	mu      sync.Mutex           // protects the following
	known   map[string]string    // map of unpacked files (real name to the temporary file name)
	handles map[string]io.Closer // map of the temporary file name to its handle
}

// newFileMgr creates a new file manager.
func newFileMgr() (*filemgr, error) {
	tmpdir, err := os.MkdirTemp("", "slackdump-*")
	if err != nil {
		return nil, err
	}
	slog.Default().Debug("created temporary directory", "dir", tmpdir)
	return &filemgr{
		tmpdir:  tmpdir,
		once:    new(sync.Once),
		known:   make(map[string]string),
		handles: make(map[string]io.Closer),
	}, nil
}

// hash returns a hex encoded sha1 hash of the string.
func hash(s string) string {
	v := sha1.Sum([]byte(s))
	return hex.EncodeToString(v[:])
}

// Destroy closes all open file handles and removes the temporary directory.
func (dp *filemgr) Destroy() error {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	var errcount int
	for hash, f := range dp.handles {
		if err := f.Close(); err != nil {
			slog.Default().Error("error closing file", "err", err)
			errcount++
			continue
		}
		delete(dp.handles, hash)
	}
	var errs error
	if errcount > 0 {
		errs = fmt.Errorf("there were %d errors closing file handles", errcount)
	}
	if err := os.RemoveAll(dp.tmpdir); err != nil {
		errs = errors.Join(errs, err)
	}
	return errs
}

// Open opens the file with the given name. If the file is already open, it
// returns the existing handle. If the file is not open, it opens the
// compressed file, unpacks it into a temporary file, and returns the handle.
// The file is expected to be a gzip-compressed file.
func (dp *filemgr) Open(name string) (*wrappedfile, error) {
	// create the directory if it doesn't exist
	var mkdirerr error
	dp.once.Do(func() {
		mkdirerr = os.MkdirAll(dp.tmpdir, 0o755)
	})
	if mkdirerr != nil {
		return nil, mkdirerr
	}

	dp.mu.Lock()
	defer dp.mu.Unlock()

	// check if the file already exists
	tmpname := hash(name)
	if tempfile, ok := dp.known[tmpname]; ok {
		f, err := os.Open(tempfile) // existing temporary file
		if err != nil {
			return nil, err
		}
		dp.handles[tmpname] = f
		return &wrappedfile{hash: tmpname, File: f, dp: dp}, nil
	}
	// open the compressed file
	cf, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer cf.Close()
	gz, err := gzip.NewReader(cf)
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	// create a temporary file
	tf, err := os.CreateTemp(dp.tmpdir, "filemgr-*")
	if err != nil {
		return nil, err
	}
	if _, err = io.Copy(tf, gz); err != nil {
		return nil, err
	}
	if err := tf.Sync(); err != nil {
		return nil, err
	}
	if _, err := tf.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	dp.known[tmpname] = tf.Name()
	dp.handles[tmpname] = tf
	return &wrappedfile{
		hash: tmpname,
		File: tf,
		dp:   dp,
	}, nil
}

// wrappedfile is a struct that wraps an os.File and holds a reference to the
// file manager.
type wrappedfile struct {
	hash string
	*os.File
	dp *filemgr
}

// Close closes the file handle and removes it from the file manager's handles
// map.
func (wf *wrappedfile) Close() error {
	wf.dp.mu.Lock()
	defer wf.dp.mu.Unlock()
	delete(wf.dp.handles, wf.hash)
	return wf.File.Close()
}
