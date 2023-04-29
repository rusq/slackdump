package subproc

import (
	"context"
	"path/filepath"

	"github.com/rusq/slackdump/v2/downloader"
	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/structures/files"
	"github.com/slack-go/slack"
)

type DumpSubproc struct {
	baseSubproc
}

// NewDumpSubproc returns a new Dump File Subprocessor.
func NewDumpSubproc(dl *downloader.Client) DumpSubproc {
	return DumpSubproc{
		baseSubproc: baseSubproc{
			dcl: dl,
		},
	}
}

func (d DumpSubproc) Files(ctx context.Context, channel *slack.Channel, m slack.Message, ff []slack.File) error {
	for _, f := range ff {
		if !isDownloadable(&f) {
			continue
		}
		dir := chunk.ToFileID(channel.ID, m.ThreadTimestamp, true)
		filename := f.ID + "-" + f.Name
		if err := d.dcl.Download(filepath.Join(dir.String(), filename), f.URLPrivateDownload); err != nil {
			return err
		}
	}
	return nil
}

// PathUpdateFunc updates the path in URLDownload and URLPrivateDownload of every
// file in the given message slice to point to the physical downloaded file
// location.
func (d DumpSubproc) PathUpdateFunc(channelID, threadTS string, mm []slack.Message) error {
	for i := range mm {
		for j := range mm[i].Files {
			path := d.filepath(channelID, threadTS, &mm[i].Files[j])
			if err := files.UpdatePathFn(path)(&mm[i].Files[j]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d DumpSubproc) filepath(channelID, threadTS string, f *slack.File) string {
	return filepath.Join(chunk.ToFileID(channelID, threadTS, true).String(), f.ID+"-"+f.Name)
}
