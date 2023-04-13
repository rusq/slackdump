package export

import (
	"context"
	"path/filepath"

	"github.com/rusq/slackdump/v2/downloader"
	"github.com/slack-go/slack"
)

type filer interface {
	Name(f *slack.File) string
	DownloadFn(ctx context.Context, dl *downloader.Client) func(string, []slack.File) error
}

type mmfiler struct{}

func (mmfiler) Name(f *slack.File) string {
	return f.Name
}

func (mmfiler) DownloadFn(ctx context.Context, dl *downloader.Client) func(string, []slack.File) error {
	const baseDir = "__uploads"
	return func(chID string, ff []slack.File) error {
		for _, f := range ff {
			if _, err := dl.DownloadFile(filepath.Join(baseDir, f.ID), f); err != nil {
				return err
			}
		}
		return nil
	}
}

type nopfiler struct{}

func (nopfiler) Name(f *slack.File) string {
	return ""
}

func (nopfiler) DownloadFn(ctx context.Context, dl *downloader.Client) func(string, []slack.File) error {
	return func(chID string, ff []slack.File) error {
		return nil
	}
}
