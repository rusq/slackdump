package fileproc

import (
	"context"
	"fmt"
	"path"
	"path/filepath"

	"github.com/rusq/slackdump/v3/internal/chunk"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk/transform"
	"github.com/rusq/slackdump/v3/processor"
)

// NewExport initialises an export file subprocessor based on the given export
// type.  This subprocessor can be later plugged into the
// [expproc.Conversations] processor.
func NewExport(typ StorageType, dl Downloader) processor.Filer {
	switch typ {
	case STstandard:
		return NewWithPathFn(dl, StdFilepath)
	case STmattermost:
		return NewWithPathFn(dl, MattermostFilepath)
	default:
		return nopsubproc{}
	}
}

// New creates a new file processor that uses mattermost file naming
// pattern.
func New(dl Downloader) processor.Filer {
	return NewWithPathFn(dl, MattermostFilepath)
}

// MattermostFilepath returns the path to the file within the __uploads
// directory.
func MattermostFilepath(_ *slack.Channel, f *slack.File) string {
	return filepath.Join(chunk.UploadsDir, f.ID, f.Name)
}

// MattermostFilepathWithDir returns the path to the file within the given
// directory, but it follows the mattermost naming pattern.
func MattermostFilepathWithDir(dir string) func(*slack.Channel, *slack.File) string {
	return func(_ *slack.Channel, f *slack.File) string {
		return path.Join(dir, f.ID, f.Name)
	}
}

func StdFilepath(ci *slack.Channel, f *slack.File) string {
	return path.Join(transform.ExportChanName(ci), "attachments", fmt.Sprintf("%s-%s", f.ID, f.Name))
}

// nopsubproc is the no-op subprocessor.
type nopsubproc struct{}

func (nopsubproc) Files(ctx context.Context, channel *slack.Channel, _ slack.Message, ff []slack.File) error {
	return nil
}

func (nopsubproc) Close() error {
	return nil
}
