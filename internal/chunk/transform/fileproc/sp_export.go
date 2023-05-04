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
func NewExport(typ StorageType, dl Downloader) processor.Filer {
	switch typ {
	case STStandard:
		return stdsubproc{
			baseSubproc: baseSubproc{
				dcl: dl,
			},
		}
	case STMattermost:
		return mmsubproc{
			baseSubproc: baseSubproc{
				dcl: dl,
			},
		}
	default:
		return nopsubproc{}
	}
}

// mmsubproc is the mattermost subprocessor
type mmsubproc struct {
	baseSubproc
}

func (mm mmsubproc) Files(ctx context.Context, channel *slack.Channel, _ slack.Message, ff []slack.File) error {
	const baseDir = "__uploads"
	for _, f := range ff {
		if !isDownloadable(&f) {
			continue
		}
		if err := mm.dcl.Download(filepath.Join(baseDir, f.ID, f.Name), f.URLPrivateDownload); err != nil {
			return err
		}
	}
	return nil
}

// stdsubproc is the standard subprocessor.
type stdsubproc struct {
	baseSubproc
}

func (mm stdsubproc) Files(ctx context.Context, channel *slack.Channel, _ slack.Message, ff []slack.File) error {
	const baseDir = "attachments"
	for _, f := range ff {
		if !isDownloadable(&f) {
			continue
		}
		if err := mm.dcl.Download(
			filepath.Join(transform.ExportChanName(channel), baseDir, fmt.Sprintf("%s-%s", f.ID, f.Name)),
			f.URLPrivateDownload,
		); err != nil {
			return err
		}
	}
	return nil
}

// nopsubproc is the no-op subprocessor.
type nopsubproc struct{}

func (nopsubproc) Files(ctx context.Context, channel *slack.Channel, _ slack.Message, ff []slack.File) error {
	return nil
}
