package export

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

type mattermostDownload struct {
	baseDownloader
}

func newMattermostDl(fs fsadapter.FS, cl *slack.Client, l logger.Interface) *mattermostDownload {
	return &mattermostDownload{baseDownloader: baseDownloader{
		dl: downloader.New(cl, fs, downloader.Logger(l), downloader.WithNameFunc(
			func(f *slack.File) string {
				return f.Name
			},
		)), l: l,
	}}
}

func (md *mattermostDownload) ProcessFunc(_ string) slackdump.ProcessFunc {
	const (
		baseDir = "__uploads"
	)
	return func(msg []types.Message, channelID string) (slackdump.ProcessResult, error) {
		total := 0
		if err := files.Extract(msg, files.Root, func(file slack.File, addr files.Addr) error {
			filedir := filepath.Join(baseDir, file.ID)
			_, err := md.dl.DownloadFile(filedir, file)
			if err != nil {
				return err
			}
			total++
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
