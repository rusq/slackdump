package export

import (
	"context"
	"errors"
	"net/url"
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

// fileProcessor is the file exporter interface.
type fileProcessor interface {
	// ProcessFunc returns the process function that should be passed to
	// DumpMessagesRaw. It should be able to extract files from the messages
	// and download them.  If the downloader is not started, i.e. if file
	// download is disabled, it should silently ignore the error and return
	// nil.
	ProcessFunc(channelName string) slackdump.ProcessFunc
}

type fileExporter interface {
	fileProcessor
	Start(ctx context.Context)
	Stop()
}

type baseDownloader struct {
	dl    *downloader.Client
	l     logger.Interface
	token string // token is the token that will be appended to each file URL.
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

// newFileExporter returns the appropriate file exporter for the ExportType.
func newFileExporter(t ExportType, fs fsadapter.FS, cl *slack.Client, l logger.Interface, token string) fileExporter {
	switch t {
	case TNoDownload:
		return newFileUpdater(token)
	default:
		l.Printf("unknown export type %s, using standard format", t)
		fallthrough
	case TStandard:
		return newStdDl(fs, cl, l, token)
	case TMattermost:
		return newMattermostDl(fs, cl, l, token)
	}
}

// newStdDl returns standard downloader, which downloads files into
// "channel_id/attachments" directory.
func newStdDl(fs fsadapter.FS, cl *slack.Client, l logger.Interface, token string) *stdDownload {
	return &stdDownload{
		baseDownloader: baseDownloader{
			dl:    downloader.New(cl, fs, downloader.Logger(l)),
			l:     l,
			token: token,
		}}
}

// ProcessFunc returns the function that downloads the file into
// channel_id/attachments directory. If Slack token is set, it updates the
// thumbnails to include that token.  It replaces the file URL to point to
// physical downloaded files on disk.
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
			if d.token != "" {
				files.Update(msg, addr, updateTokenFn(d.token))
			}
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

// newMattermostDl returns the downloader, that downloads the files into
// the __uploads directory, so that it could be transformed into bulk import
// by mmetl and imported into mattermost with mmctl import bulk.
func newMattermostDl(fs fsadapter.FS, cl *slack.Client, l logger.Interface, token string) *mattermostDownload {
	return &mattermostDownload{
		baseDownloader: baseDownloader{
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
func (md *mattermostDownload) ProcessFunc(_ string) slackdump.ProcessFunc {
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
				return files.Update(msgs, addr, updateTokenFn(md.token))
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

// addToken updates the uri, adding the t= query parameter with token value.
// if token or url is empty, it does nothing.
func addToken(uri string, token string) (string, error) {
	if token == "" || uri == "" {
		return uri, nil
	}
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	val := u.Query()
	val.Set("t", token)
	u.RawQuery = val.Encode()
	return u.String(), nil
}

// updateTokenFn returns a file update function that adds the t= query parameter
// with token value. If token value is empty, the function does nothing.
func updateTokenFn(token string) func(*slack.File) error {
	return func(f *slack.File) error {
		if token == "" {
			return nil
		}
		var err error
		update := func(s *string, t string) {
			if err != nil {
				return
			}
			*s, err = addToken(*s, t)
		}
		update(&f.URLPrivate, token)
		update(&f.URLPrivateDownload, token)
		update(&f.Thumb64, token)
		update(&f.Thumb80, token)
		update(&f.Thumb160, token)
		update(&f.Thumb360, token)
		update(&f.Thumb360Gif, token)
		update(&f.Thumb480, token)
		update(&f.Thumb720, token)
		update(&f.Thumb960, token)
		update(&f.Thumb1024, token)
		return nil
	}
}

// fileUpdater does not download any files, it just updates the link adding
// a token query parameter, if the token is set.
type fileUpdater struct {
	baseDownloader
}

// Start does nothing.
func (fileUpdater) Start(context.Context) {}

// Stop does nothing.
func (fileUpdater) Stop() {}

// newFileUpdater returns an fileExporter that does not download any files,
// but updates the link adding a token query parameter, if the token is set.
func newFileUpdater(token string) fileUpdater {
	return fileUpdater{baseDownloader: baseDownloader{
		token: token,
	}}
}

// ProcessFunc returns the [slackdump.ProcessFunc] that updates the file link
// adding a token query parameter.
func (u fileUpdater) ProcessFunc(_ string) slackdump.ProcessFunc {
	if u.token == "" {
		return func(msg []types.Message, channelID string) (slackdump.ProcessResult, error) {
			return slackdump.ProcessResult{}, nil
		}
	}
	return func(msgs []types.Message, channelID string) (slackdump.ProcessResult, error) {
		total := 0
		if err := files.Extract(msgs, files.Root, func(file slack.File, addr files.Addr) error {
			return files.Update(msgs, addr, updateTokenFn(u.token))
		}); err != nil {
			return slackdump.ProcessResult{}, err
		}
		return slackdump.ProcessResult{Entity: entFiles, Count: total}, nil
	}
}
