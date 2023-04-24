package transform

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/downloader"
	"github.com/rusq/slackdump/v2/export"
	"github.com/rusq/slackdump/v2/internal/chunk/processor"
	"github.com/rusq/slackdump/v2/internal/structures/files"
)

// Filer initialises a filer type based on the given export type.
// This filer can be later plugged into the Conversations processor using
// WithFiler option.
func NewFiler(typ export.ExportType, dl *downloader.Client) processor.Filer {
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

type baseSubproc struct {
	dcl *downloader.Client
}

type mmsubproc struct {
	baseSubproc
}

func (mm mmsubproc) Files(ctx context.Context, channel *slack.Channel, _ slack.Message, _ bool, ff []slack.File) error {
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

type nopsubproc struct{}

func (nopsubproc) Files(ctx context.Context, channel *slack.Channel, _ slack.Message, _ bool, ff []slack.File) error {
	return nil
}

type stdsubproc struct {
	baseSubproc
}

func (mm stdsubproc) Files(ctx context.Context, channel *slack.Channel, _ slack.Message, _ bool, ff []slack.File) error {
	const baseDir = "attachments"
	for _, f := range ff {
		if !isDownloadable(&f) {
			continue
		}
		if err := mm.dcl.Download(
			filepath.Join(channelName(channel), baseDir, fmt.Sprintf("%s-%s", f.ID, f.Name)),
			f.URLPrivateDownload,
		); err != nil {
			return err
		}
	}
	return nil
}

// ExportTokenUpdateFn returns a function that appends the token to every file
// URL in the given message.
func ExportTokenUpdateFn(token string) func(msg *slack.Message) error {
	fn := files.UpdateTokenFn(token)
	return func(msg *slack.Message) error {
		for i := range msg.Files {
			if err := fn(&msg.Files[i]); err != nil {
				return err
			}
		}
		return nil
	}
}

type dumpSubproc struct {
	baseSubproc
}

func NewDumpSubproc(dl *downloader.Client) processor.Filer {
	return dumpSubproc{
		baseSubproc: baseSubproc{
			dcl: dl,
		},
	}
}

func (d dumpSubproc) Files(ctx context.Context, channel *slack.Channel, _ slack.Message, _ bool, ff []slack.File) error {
	for _, f := range ff {
		if !isDownloadable(&f) {
			continue
		}
		filename := f.ID + "-" + f.Name
		if err := d.dcl.Download(filepath.Join(channel.ID, filename), f.URLPrivateDownload); err != nil {
			return err
		}
	}
	return nil
}

// isDownloadable returns true if the file can be downloaded.
func isDownloadable(f *slack.File) bool {
	return f.Mode != "hidden_by_limit" && f.Mode != "external" && !f.IsExternal
}
