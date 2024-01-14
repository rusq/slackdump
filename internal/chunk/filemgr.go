package chunk

import (
	"compress/gzip"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/rusq/slackdump/v3/logger"
)

const extIdx = ".idx"

type filemgr struct {
	tmpdir string // temporary storage directory

	mu       sync.Mutex           // protects the following
	existing map[string]string    // map of unpacked files (real name to the temporary file name)
	handles  map[string]io.Closer // map of the temporary file name to it's handle
}

func newFileMgr() (*filemgr, error) {
	tmpdir, err := os.MkdirTemp("", "slackdump-*")
	if err != nil {
		return nil, err
	}
	logger.Default.Debugf("created temporary directory: %s", tmpdir)
	return &filemgr{
		tmpdir:   tmpdir,
		existing: make(map[string]string),
		handles:  make(map[string]io.Closer),
	}, nil
}

// hash returns a hex encoded sha256 hash of the string.
func hash(s string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(s)))
}

func (dp *filemgr) Destroy() error {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	for _, f := range dp.handles {
		f.Close()
	}
	return os.RemoveAll(dp.tmpdir)
}

// Open
func (dp *filemgr) Open(name string) (*wrappedfile, error) {
	// create the directory if it doesn't exist
	if err := os.MkdirAll(dp.tmpdir, 0o755); err != nil {
		return nil, err
	}
	// check if the file is already open
	dp.mu.Lock()
	defer dp.mu.Unlock()
	if tempfile, ok := dp.existing[name]; ok {
		etf, err := os.Open(tempfile) // existing temporary file
		if err != nil {
			return nil, err
		}
		dp.handles[etf.Name()] = etf
		return &wrappedfile{etf, dp}, nil
	}
	// open the compressed file
	tmpname := hash(name)
	cf, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer cf.Close()
	gz, err := gzip.NewReader(cf)
	if err != nil {
		return nil, err
	}
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
	dp.existing[name] = tf.Name()
	dp.handles[tmpname] = tf
	return &wrappedfile{tf, dp}, nil
}

type wrappedfile struct {
	*os.File
	dp *filemgr
}

func (wf *wrappedfile) Close() error {
	wf.dp.mu.Lock()
	defer wf.dp.mu.Unlock()
	delete(wf.dp.handles, wf.Name())
	return wf.File.Close()
}
