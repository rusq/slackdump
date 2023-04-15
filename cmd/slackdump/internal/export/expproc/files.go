package expproc

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/rusq/slackdump/v2/downloader"
	"github.com/rusq/slackdump/v2/export"
	"github.com/rusq/slackdump/v2/internal/chunk/processor"
	"github.com/slack-go/slack"
)

// Filer initialises a filer type based on the given export type.
// This filer can be later plugged into the Conversations processor using
// WithFiler option.
func NewFiler(typ export.ExportType, dl *downloader.Client) processor.Filer {
	switch typ {
	case export.TStandard:
		return stdfiler{
			basefiler: basefiler{
				dcl: dl,
			},
		}
	case export.TMattermost:
		return mmfiler{
			basefiler: basefiler{
				dcl: dl,
			},
		}
	default:
		return nopfiler{}
	}
}

type basefiler struct {
	dcl *downloader.Client
}

type mmfiler struct {
	basefiler
}

func (mm mmfiler) Files(ctx context.Context, channelID string, parent slack.Message, isThread bool, ff []slack.File) error {
	const baseDir = "__uploads"
	for _, f := range ff {
		if err := mm.dcl.Download(filepath.Join(baseDir, f.ID, f.Name), f.URLPrivateDownload); err != nil {
			return err
		}
	}
	return nil
}

type nopfiler struct{}

func (nopfiler) Files(ctx context.Context, channelID string, parent slack.Message, isThread bool, ff []slack.File) error {
	return nil
}

type stdfiler struct {
	basefiler
}

func (mm stdfiler) Files(ctx context.Context, channelID string, parent slack.Message, isThread bool, ff []slack.File) error {
	const baseDir = "attachments"
	for _, f := range ff {
		if err := mm.dcl.Download(
			filepath.Join(channelID, baseDir, fmt.Sprintf("%s-%s", f.ID, f.Name)),
			f.URLPrivateDownload,
		); err != nil {
			return err
		}
	}
	return nil
}
