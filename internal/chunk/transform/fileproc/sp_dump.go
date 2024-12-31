package fileproc

import (
	"path"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
)

// NewDumpSubproc returns a new Dump File Subprocessor.
func NewDumpSubproc(dl Downloader) Subprocessor {
	return Subprocessor{
		dcl:      dl,
		filepath: DumpFilepath,
	}
}

func DumpFilepath(ci *slack.Channel, f *slack.File) string {
	return path.Join(chunk.ToFileID(ci.ID, "", false).String(), f.ID+"-"+f.Name)
}
