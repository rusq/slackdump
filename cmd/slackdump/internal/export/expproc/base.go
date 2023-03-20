package expproc

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rusq/slackdump/v2/internal/chunk"
)

type baseproc struct {
	dir string
	wf  io.WriteCloser // processor recording
	*chunk.Recorder
}

func newBaseProc(dir string, filename string) (*baseproc, error) {
	if fi, err := os.Stat(dir); err != nil {
		return nil, err
	} else if !fi.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", dir)
	}
	wf, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		return nil, err
	}
	r := chunk.NewRecorder(wf)
	return &baseproc{dir: dir, wf: wf, Recorder: r}, nil
}

func (p *baseproc) Close() error {
	if err := p.Recorder.Close(); err != nil {
		p.wf.Close()
		return err
	}
	return p.wf.Close()
}
