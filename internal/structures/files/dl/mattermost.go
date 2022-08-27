package dl

// mattermost file format support

import (
	"errors"
	"path/filepath"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/downloader"
	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/rusq/slackdump/v2/internal/structures/files"
	"github.com/rusq/slackdump/v2/logger"
	"github.com/rusq/slackdump/v2/types"
)

type Mattermost struct {
	base
}

// NewMattermost returns the dl, that downloads the files into
// the __uploads directory, so that it could be transformed into bulk import
// by mmetl and imported into mattermost with mmctl import bulk.
func NewMattermost(fs fsadapter.FS, cl *slack.Client, l logger.Interface, token string) *Mattermost {
	return &Mattermost{
		base: base{
			l:     l,
			token: token,
			dl: downloader.New(cl, fs, downloader.Logger(l), downloader.WithNameFunc(
				func(f *slack.File) string {
					return f.Name
				},
			)),
		},
	}
}

// ProcessFunc returns the ProcessFunc that downloads the files into the
// __uploads directory in the root of the download filesystem.
func (md *Mattermost) ProcessFunc(_ string) slackdump.ProcessFunc {
	const (
		baseDir = "__uploads"
	)
	return func(msgs []types.Message, channelID string) (slackdump.ProcessResult, error) {
		total := 0
		if err := files.Extract(msgs, files.Root, func(file slack.File, addr files.Addr) error {
			filedir := filepath.Join(baseDir, file.ID)
			_, err := md.dl.DownloadFile(filedir, file)
			if err != nil {
				return err
			}
			total++
			if md.token != "" {
				return files.Update(msgs, addr, files.UpdateTokenFn(md.token))
			}
			return nil
		}); err != nil {
			if errors.Is(err, downloader.ErrNotStarted) {
				return slackdump.ProcessResult{Entity: entFiles, Count: 0}, nil
			}
			return slackdump.ProcessResult{}, err
		}
		return slackdump.ProcessResult{Entity: entFiles, Count: total}, nil
	}
}
