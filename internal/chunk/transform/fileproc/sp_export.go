package fileproc

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/rusq/slackdump/v2/export"
	"github.com/rusq/slackdump/v2/internal/chunk/transform"
	"github.com/rusq/slackdump/v2/processor"
	"github.com/slack-go/slack"
)

// NewExport initialises an export file subprocessor based on the given export
// type.  This subprocessor can be later plugged into the
// [expproc.Conversations] processor.
func NewExport(typ export.ExportType, dl Downloader) processor.Filer {
	switch typ {
	case export.TStandard:
		return stdsubproc{
			baseSubproc: baseSubproc{
				dcl: dl,
			},
		}
	case export.TMattermost:
		return mmsubproc{
			baseSubproc: baseSubproc{
				dcl: dl,
			},
		}
	default:
		return nopsubproc{}
	}
}

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

type nopsubproc struct{}

func (nopsubproc) Files(ctx context.Context, channel *slack.Channel, _ slack.Message, ff []slack.File) error {
	return nil
}

type stdsubproc struct {
	baseSubproc
}
