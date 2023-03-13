package transform

import (
	"archive/zip"
	"fmt"
	"path"

	"github.com/rusq/slackdump/v2/export"
	"github.com/rusq/slackdump/v2/internal/chunk/state"
)

type Export struct{}

func NewExport() *Export { return &Export{} }

type ExportInfo struct {
	IsComplete bool
	State      *state.State
	ExportType export.ExportType
	Filename   string
}

func (e *Export) RestoreState(loc string) (*state.State, error) {
	zf, err := zip.OpenReader(loc)
	if err != nil {
		return nil, err
	}
	defer zf.Close()

	var ei = ExportInfo{
		State: state.New(""),
	}

	for _, f := range zf.File {
		switch path.Base(f.Name) {
		case "channels.json":
		case "mpims.json":
		case "dms.json":
		case "users.jsone":
			ei.IsComplete = true
		default:
			// TODO: handle other files
		}
		fmt.Println(path.Split(f.Name))
	}

	return nil, nil
}
