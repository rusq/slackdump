// Package redownload contains redownload logic.
package redownload

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/convert/transform/fileproc"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/processor"
	"github.com/rusq/slackdump/v3/source"
)

type Redownloader struct {
	// src is the source we operate on
	src source.SourceResumeCloser
	// flags are the flags of the source
	flags source.Flags

	// dir is the path to the source.
	dir string

	// druRun if set to true, we don't download anything, just count.
	dryRun bool
}

func New(ctx context.Context, dir string, dry bool) (*Redownloader, error) {
	if err := validate(dir); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}
	flags, err := source.Type(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to determine type: %w", err)
	}
	src, err := source.Load(ctx, dir)
	if err != nil {
		return nil, fmt.Errorf("error opening source data: %w", err)
	}
	return &Redownloader{
		src:    src,
		flags:  flags,
		dir:    dir,
		dryRun: dry,
	}, nil
}

func (r *Redownloader) Stop() error {
	return r.src.Close()
}

// validate ensures that the directory is a Slackdump Archive directory.
// It sets the exit status according to the error type.
func validate(dir string) error {
	flags, err := source.Type(dir)
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return fmt.Errorf("error determining source type: %w", err)
	}
	if flags&source.FZip != 0 {
		base.SetExitStatus(base.SUserError)
		return errors.New("unable to work with ZIP files, unpack it first")
	}

	return nil
}

// pathFunc returns effective path function.
func (r *Redownloader) pathFunc() func(ch *slack.Channel, f *slack.File) string {
	if r.src.Files().Type() != source.STnone {
		// easy
		return r.src.Files().FilePath
	}
	switch {
	case r.flags&source.FDump != 0:
		return source.DumpFilepath
	default:
		// in all other cases we default to mattermost file path.
		return source.MattermostFilepath
	}
	// unreachable
}

// fileProcessorForSource returns the appropriate file processor.
func (r *Redownloader) fileProc(dl fileproc.Downloader) (processor.Filer, error) {
	var fproc processor.Filer
	fl := r.flags
	switch {
	case fl&source.FDatabase != 0 || fl&source.FChunk != 0:
		fproc = fileproc.New(dl)
	case fl&source.FExport != 0:
		typ := r.src.Files().Type()
		if typ == source.STnone {
			typ = source.STmattermost // default to mattermost
		}
		fproc = fileproc.NewExport(typ, dl)
	case fl&source.FDump != 0:
		fproc = fileproc.NewDump(dl)
	default:
		return nil, fmt.Errorf("unable to determine file storage format for the source with flags %s", fl)
	}
	return fproc, nil
}

// FileStats contains the file statistics.
type FileStats struct {
	NumFiles uint
	NumBytes uint64
}

func (r *Redownloader) channels(ctx context.Context) ([]slack.Channel, error) {
	channels, err := r.src.Channels(ctx)
	if err != nil {
		return nil, fmt.Errorf("error reading channels: %w", err)
	}
	if len(channels) == 0 {
		return nil, errors.New("no channels found")
	}
	return channels, nil
}

func (r *Redownloader) Stats(ctx context.Context) (FileStats, error) {
	var ret FileStats

	channels, err := r.channels(ctx)
	if err != nil {
		return ret, err
	}

	for _, ch := range channels {
		items, err := r.scanChannel(ctx, &ch)
		if err != nil {
			return ret, err
		}
		for _, item := range items {
			ret.NumFiles++
			ret.NumBytes += uint64(item.f.Size)
		}
	}

	return ret, nil
}

func (r *Redownloader) Download(ctx context.Context) (FileStats, error) {
	var ret FileStats
	channels, err := r.channels(ctx)
	if err != nil {
		return ret, err
	}

	client, err := bootstrap.Slack(ctx)
	if err != nil {
		return ret, fmt.Errorf("error creating slackdump session: %w", err)
	}
	dl := fileproc.NewDownloader(
		ctx,
		true,
		client,
		fsadapter.NewDirectory(r.src.Name()),
		cfg.Log,
	)
	defer dl.Stop()

	// determine the file processor for the source.
	fproc, err := r.fileProc(dl)
	if err != nil {
		return ret, err
	}
	defer fproc.Close()

	for _, ch := range channels {
		items, err := r.scanChannel(ctx, &ch)
		if err != nil {
			return ret, err
		}
		for _, item := range items {
			if err := fproc.Files(ctx, item.ch, *item.msg, []slack.File{*item.f}); err != nil {
				return ret, err
			}
			ret.NumFiles++
			ret.NumBytes += uint64(item.f.Size)
		}
	}
	return ret, nil
}

type dlItem struct {
	msg *slack.Message
	f   *slack.File
	ch  *slack.Channel
}

func (r *Redownloader) scanChannel(ctx context.Context, ch *slack.Channel) ([]dlItem, error) {
	slog.Info("scanning channel", "channel", ch.ID)
	it, err := r.src.AllMessages(ctx, ch.ID)
	if err != nil {
		if errors.Is(err, source.ErrNotFound) {
			// no data in the channel
			return nil, nil
		}
		return nil, fmt.Errorf("error reading messages: %w", err)
	}
	// collect messages from the iterator
	msgs, err := collect(it)
	if err != nil {
		return nil, fmt.Errorf("error fetching messages: %w", err)
	}

	if len(msgs) == 0 {
		return nil, nil
	}
	slog.Info("scanning messages", "num_messages", len(msgs))
	return r.scanMsgs(ctx, ch, msgs, false)
}

// collect collects all Ks from iterator it, returning any encountered error.
func collect[K any](it iter.Seq2[K, error]) ([]K, error) {
	kk := make([]K, 0)
	for k, err := range it {
		if err != nil {
			return kk, fmt.Errorf("error fetching messages: %w", err)
		}
		kk = append(kk, k)
	}
	return kk, nil
}

func (r *Redownloader) scanMsgs(ctx context.Context, ch *slack.Channel, msgs []slack.Message, isThread bool) ([]dlItem, error) {
	lg := slog.With("channel", ch.ID)
	// workaround for completely missing storage
	pathFn := r.pathFunc()
	var toDl []dlItem
	for _, m := range msgs {
		if structures.IsThreadStart(&m) && !isThread {
			it, err := r.src.AllThreadMessages(ctx, ch.ID, m.ThreadTimestamp)
			if err != nil {
				return nil, fmt.Errorf("error reading thread messages: %w", err)
			}
			tm, err := collect(it)
			if err != nil {
				return nil, fmt.Errorf("error collecting thread messages: %w", err)
			}

			lg.Info("scanning thread messages", "num_messages", len(tm), "thread", m.ThreadTimestamp)
			if res, err := r.scanMsgs(ctx, ch, tm, true); err != nil {
				return toDl, err
			} else {
				toDl = append(toDl, res...)
			}
		}

		// collect all missing files from the message.
		var missing []slack.File
		for _, ff := range m.Files {
			if !fileproc.IsValid(&ff) {
				lg.Debug("file is not valid for download", "ID", ff.ID)
				continue
			}

			name := filepath.Join(r.src.Name(), pathFn(ch, &ff))
			lg := lg.With("file", name)
			lg.Debug("checking file")

			if fi, err := os.Stat(name); err != nil {
				if os.IsNotExist(err) {
					// file does not exist
					lg.Debug("missing file")
					missing = append(missing, ff)
				} else {
					lg.Error("error accessing file", "error", err)
					// some other error
					return toDl, fmt.Errorf("error accessing file: %w", err)
				}
			} else if fi.Size() == 0 {
				// zero length files are considered missing
				lg.Debug("zero length file")
				missing = append(missing, ff)
			} else {
				lg.Debug("file OK")
			}
		}

		for _, f := range missing {
			toDl = append(toDl, dlItem{
				msg: &m,
				ch:  ch,
				f:   &f,
			})
		}
	}
	return toDl, nil
}
