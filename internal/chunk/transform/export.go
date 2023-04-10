package transform

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v2/export"
	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/chunk/state"
)

type Export struct {
	fsa fsadapter.FS
}

func NewExport(fsa fsadapter.FS) *Export { return &Export{fsa: fsa} }

type ExportInfo struct {
	IsComplete bool
	State      *state.State
	ExportType export.ExportType
	Filename   string
}

func ExportState(loc string) (*state.State, error) {
	zf, err := zip.OpenReader(loc)
	if err != nil {
		return nil, err
	}
	defer zf.Close()

	ei := ExportInfo{
		State: state.New(""),
	}

	for _, f := range zf.File {
		switch path.Base(f.Name) {
		case "channels.json":
		case "mpims.json":
		case "dms.json":
		case "users.json":
			ei.IsComplete = true
		default:
			// TODO: handle other files
		}
		fmt.Println(path.Split(f.Name))
	}

	return nil, nil
}

func loadState(st *state.State, basePath string) (io.ReadSeekCloser, error) {
	if st == nil {
		return nil, fmt.Errorf("fatal:  nil state")
	}
	if !st.IsComplete {
		return nil, fmt.Errorf("fatal:  incomplete state")
	}
	rsc, err := st.OpenChunks(basePath)
	if err != nil {
		return nil, err
	}
	return rsc, nil
}

func (e *Export) Transform(ctx context.Context, basePath string, st *state.State) error {
	// check if the base directory exists
	if err := checkDir(basePath); err != nil {
		return err
	}

	rsc, err := loadState(st, basePath)
	if err != nil {
		return err
	}
	defer rsc.Close()

	cf, err := chunk.FromReader(rsc)
	if err != nil {
		return err
	}

	// check if it has users
	if cf.HasUsers() {
		// generate users.json
	}
	// check if it has channels
	if cf.HasChannels() {
		// generate channels.json, mpims.json, dms.json
	}
	// check if it has conversations
	channels := cf.AllChannelIDs()
	if len(channels) != 0 {
		// process channels
	}

	return nil
}

func checkDir(dir string) error {
	if fi, err := os.Stat(dir); err != nil {
		return err
	} else if !fi.IsDir() {
		return fmt.Errorf("%q is not a directory", dir)
	}
	return nil
}
