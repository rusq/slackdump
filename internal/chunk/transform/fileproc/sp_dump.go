package fileproc

import (
	"path"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
)

// NewDump returns a new Dump File FileProcessor.
func NewDump(dl Downloader) FileProcessor {
	return New(dl, DumpFilepath)
}

func DumpFilepath(ci *slack.Channel, f *slack.File) string {
	return path.Join(chunk.ToFileID(ci.ID, "", false).String(), f.ID+"-"+f.Name)
}
