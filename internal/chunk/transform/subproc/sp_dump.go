package subproc

import (
	"context"
	"path/filepath"

	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/structures/files"
	"github.com/slack-go/slack"
)

// DumpSubproc is a file subprocessor that downloads all files to the local
// filesystem using underlying downloader.
type DumpSubproc struct {
	baseSubproc
}

// NewDumpSubproc returns a new Dump File Subprocessor.
func NewDumpSubproc(dl Downloader) DumpSubproc {
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
		if err := d.dcl.Download(d.filepath(channel.ID, &f), f.URLPrivateDownload); err != nil {
			return err
		}
	}
	return nil
}

// PathUpdateFunc updates the path in URLDownload and URLPrivateDownload of every
// file in the given message slice to point to the physical downloaded file
// location.  It can be plugged in the pipeline of Dump.
func (d DumpSubproc) PathUpdateFunc(channelID, threadTS string, mm []slack.Message) error {
	for i := range mm {
		for j := range mm[i].Files {
			path := d.filepath(channelID, &mm[i].Files[j])
			if err := files.UpdatePathFn(path)(&mm[i].Files[j]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d DumpSubproc) filepath(channelID string, f *slack.File) string {
	return filepath.Join(chunk.ToFileID(channelID, "", false).String(), f.ID+"-"+f.Name)
}
