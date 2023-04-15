package export

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/rusq/slackdump/v2/downloader"
	"github.com/rusq/slackdump/v2/export"
	"github.com/slack-go/slack"
)

type filer interface {
	Download(ctx context.Context, channelID string, ff []slack.File) error
}

func filerForType(typ export.ExportType) filer {
	switch typ {
	case export.TStandard:
		return stdfiler{}
	case export.TMattermost:
		return mmfiler{}
	default:
		return nopfiler{}
	}
}

type basefiler struct {
	dcl *downloader.ClientV2
}

type mmfiler struct {
	basefiler
}

func (mm mmfiler) Download(ctx context.Context, channelID string, ff []slack.File) error {
	const baseDir = "__uploads"
	for _, f := range ff {
		if err := mm.dcl.Download(filepath.Join(baseDir, f.ID, f.Name), f.URLPrivateDownload); err != nil {
			return err
		}
	}
	return nil
}

type nopfiler struct{}

func (nopfiler) Download(ctx context.Context, url string, ff []slack.File) error {
	return nil
}

type stdfiler struct {
	basefiler
}

func (mm stdfiler) Download(ctx context.Context, channelID string, ff []slack.File) error {
	const baseDir = "attachments"
	for _, f := range ff {
		if err := mm.dcl.Download(
			filepath.Join(channelID, baseDir, fmt.Sprintf("%s-%s", f.ID, f.Name)),
			f.URL,
		); err != nil {
			return err
		}
	}
	return nil
}
