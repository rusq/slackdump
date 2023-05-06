package fileproc

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/rusq/slackdump/v2/internal/chunk/transform"
	"github.com/rusq/slackdump/v2/processor"
	"github.com/slack-go/slack"
)

// NewExport initialises an export file subprocessor based on the given export
// type.  This subprocessor can be later plugged into the
// [expproc.Conversations] processor.
func NewExport(typ StorageType, dl Downloader) processor.Files {
	switch typ {
	case STStandard:
		return Subprocessor{
			dcl:      dl,
			filepath: StdFilepath,
		}
	case STMattermost:
		return Subprocessor{
			dcl:      dl,
			filepath: MattermostFilepath,
		}
	default:
		return nopsubproc{}
	}
}

func MattermostFilepath(_ *slack.Channel, f *slack.File) string {
	return filepath.Join("__uploads", f.ID, f.Name)
}

func StdFilepath(ci *slack.Channel, f *slack.File) string {
	return filepath.Join(transform.ExportChanName(ci), "attachments", fmt.Sprintf("%s-%s", f.ID, f.Name))
}

// nopsubproc is the no-op subprocessor.
type nopsubproc struct{}

func (nopsubproc) Files(ctx context.Context, channel *slack.Channel, _ slack.Message, ff []slack.File) error {
	return nil
}
