// Package fileproc is the file processor that can be used in conjunction with
// the transformer.  It downloads files to the local filesystem using the
// provided downloader.  Probably it's a good idea to use the
// [downloader.Client] for this.
package fileproc

import (
	"context"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v2/downloader"
	"github.com/rusq/slackdump/v2/internal/structures/files"
	"github.com/rusq/slackdump/v2/logger"
	"github.com/slack-go/slack"
)

// Downloader is the interface that wraps the Download method.
type Downloader interface {
	// Download should download the file at the given URL and save it to the
	// given path.
	Download(fullpath string, url string) error
}

// Subprocessor is the file subprocessor, that downloads files to the path
// returned by the filepath function.
// Zero value of this type is not usable.
type Subprocessor struct {
	dcl      Downloader
	filepath func(ci *slack.Channel, f *slack.File) string
}

// NewSubprocessor initialises the subprocessor.
func NewSubprocessor(dl Downloader, fp func(ci *slack.Channel, f *slack.File) string) Subprocessor {
	if fp == nil {
		panic("filepath function is nil")
	}
	return Subprocessor{
		dcl:      dl,
		filepath: fp,
	}
}

func (b Subprocessor) Files(ctx context.Context, channel *slack.Channel, msg slack.Message, ff []slack.File) error {
	for _, f := range ff {
		if !IsValid(&f) {
			continue
		}
		if err := b.dcl.Download(b.filepath(channel, &f), f.URLPrivateDownload); err != nil {
			return err
		}
	}
	return nil
}

// PathUpdateFunc updates the path in URLDownload and URLPrivateDownload of every
// file in the given message slice to point to the physical downloaded file
// location.  It can be plugged in the pipeline of Dump.
func (b Subprocessor) PathUpdateFunc(channelID, threadTS string, mm []slack.Message) error {
	for i := range mm {
		for j := range mm[i].Files {
			ch := new(slack.Channel)
			ch.ID = channelID
			path := b.filepath(ch, &mm[i].Files[j])
			if err := files.UpdatePathFn(path)(&mm[i].Files[j]); err != nil {
				return err
			}
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

// IsValid returns true if the file can be downloaded and is valid.
func IsValid(f *slack.File) bool {
	return f.Mode != "hidden_by_limit" && f.Mode != "external" && !f.IsExternal
}

type NoopDownloader struct{}

func (NoopDownloader) Download(fullpath string, url string) error {
	return nil
}

// NewDownloader initializes the downloader and returns it, along with a
// function that should be called to stop it.
func NewDownloader(ctx context.Context, gEnabled bool, cl downloader.Downloader, fsa fsadapter.FS, lg logger.Interface) (sdl Downloader, stop func()) {
	if !gEnabled {
		return NoopDownloader{}, func() {}
	} else {
		dl := downloader.New(cl, fsa, downloader.WithLogger(lg))
		dl.Start(ctx)
		return dl, dl.Stop
	}
}
