package fileproc

import (
	"github.com/rusq/slackdump/v3/source"
)

// NewDump returns a new Dump File FileProcessor.
func NewDump(dl Downloader) FileProcessor {
	return NewWithPathFn(dl, source.DumpFilepath)
}
