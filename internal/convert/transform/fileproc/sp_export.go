package fileproc

import (
	"github.com/rusq/slackdump/v3/internal/source"
	"github.com/rusq/slackdump/v3/processor"
)

// NewExport initialises an export file subprocessor based on the given export
// type.  This subprocessor can be later plugged into the
// [expproc.Conversations] processor.
func NewExport(typ source.StorageType, dl Downloader) processor.Filer {
	switch typ {
	case source.STstandard:
		return NewWithPathFn(dl, source.StdFilepath)
	case source.STmattermost:
		return NewWithPathFn(dl, source.MattermostFilepath)
	default:
		return &processor.NopFiler{}
	}
}

// New creates a new file processor that uses mattermost file naming
// pattern.
func New(dl Downloader) processor.Filer {
	return NewWithPathFn(dl, source.MattermostFilepath)
}
