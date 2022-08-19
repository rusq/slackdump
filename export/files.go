package export

import (
	"context"
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

const entFiles = "files"

// Downloader is the interface for the file downloader.
type Downloader interface {
	// ProcessFunc returns the process function that should be passed to
	// DumpMessagesRaw that should handle the download of the files.  If the
	// downloader is not started, i.e. if file download is disabled, it should
	// silently ignore the error and return nil.
	ProcessFunc(channelName string) slackdump.ProcessFunc
	Start(ctx context.Context)
	Stop()
}

type baseDownloader struct {
	dl *downloader.Client
	l  logger.Interface
}

func (bd *baseDownloader) Start(ctx context.Context) {
	bd.dl.Start(ctx)
}

func (bd *baseDownloader) Stop() {
	bd.dl.Stop()
}

type stdDownload struct {
	baseDownloader
}

type mattermostDownload struct {
	baseDownloader
}

func newDownloader(t ExportType, fs fsadapter.FS, cl *slack.Client, l logger.Interface) Downloader {
	switch t {
	default:
		l.Printf("unknown export type %s, using standard format", t)
		fallthrough
	case TStandard:
		return newStdDl(fs, cl, l)
	case TMattermost:
		return newMattermostDl(fs, cl, l)
	}
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
		baseDir = "__upload"
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
