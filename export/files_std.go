package export

import (
	"errors"
	"path"
	"path/filepath"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/downloader"
	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/rusq/slackdump/v2/internal/structures/files"
	"github.com/rusq/slackdump/v2/logger"
	"github.com/rusq/slackdump/v2/types"
	"github.com/slack-go/slack"
)

type stdDownload struct {
	baseDownloader
}

func newStdDl(fs fsadapter.FS, cl *slack.Client, l logger.Interface) *stdDownload {
	return &stdDownload{baseDownloader: baseDownloader{
		dl: downloader.New(cl, fs, downloader.Logger(l)),
		l:  l,
	}}
}

func (d *stdDownload) ProcessFunc(channelName string) slackdump.ProcessFunc {
	const (
		dirAttach = "attachments"
	)

	dir := filepath.Join(channelName, dirAttach)
	return func(msg []types.Message, channelID string) (slackdump.ProcessResult, error) {
		total := 0
		if err := files.Extract(msg, files.Root, func(file slack.File, addr files.Addr) error {
			filename, err := d.dl.DownloadFile(dir, file)
			if err != nil {
				return err
			}
			d.l.Debugf("submitted for download: %s", file.Name)
			total++
			return files.UpdateURLs(msg, addr, path.Join(dirAttach, path.Base(filename)))
		}); err != nil {
			if errors.Is(err, downloader.ErrNotStarted) {
				return slackdump.ProcessResult{Entity: entFiles, Count: 0}, nil
			}
			return slackdump.ProcessResult{}, err
		}

		return slackdump.ProcessResult{Entity: entFiles, Count: total}, nil
	}
}
