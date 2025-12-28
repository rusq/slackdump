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
	"runtime/trace"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/convert/transform/fileproc"
	"github.com/rusq/slackdump/v3/internal/primitive"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/processor"
	"github.com/rusq/slackdump/v3/source"
)

type Redownloader struct {
	// src is the source we operate on
	src source.SourceResumeCloser
	// flags are the flags of the source
	flags source.Flags
	lg    *slog.Logger

	// dir is the path to the source.
	dir string
}

type Option func(*Redownloader)

func WithLogger(lg *slog.Logger) Option {
	return func(r *Redownloader) {
		if lg != nil {
			r.lg = lg
		}
	}
}

// New initialises the new Redownloader for the given directory.  Source type
// is detected automatically. It validates if the source is of the supported
// type and returns any errors.
func New(ctx context.Context, dir string, opts ...Option) (*Redownloader, error) {
	st, err := source.Type(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to determine type: %w", err)
	}
	if err := validate(st); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}
	src, err := source.Load(ctx, dir)
	if err != nil {
		return nil, fmt.Errorf("error opening source data: %w", err)
	}
	r := &Redownloader{
		src:   src,
		flags: st,
		dir:   dir,
		lg:    slog.Default(),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r, nil
}

// Stop stops the Redownloader.
func (r *Redownloader) Stop() error {
	return r.src.Close()
}

// validate ensures that the directory is a Slackdump Archive directory.
// It sets the exit status according to the error type.
func validate(st source.Flags) error {
	if st&source.FZip != 0 {
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

	// no existing file storage directory.
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
		storageType := r.src.Files().Type()
		storageType = primitive.IfTrue(storageType == source.STnone, source.STmattermost, storageType)
		fproc = fileproc.NewExport(storageType, dl)
	case fl&source.FDump != 0:
		fproc = fileproc.NewDump(dl)
	default:
		return nil, fmt.Errorf("unable to determine file storage format for the source with flags %s", fl)
	}
	return fproc, nil
}

var ErrNoChannels = errors.New("no channels found")

// channels returns the channels in the underlying source. It returns
// ErrNoChannels if there are zero channels.
func (r *Redownloader) channels(ctx context.Context) ([]slack.Channel, error) {
	channels, err := r.src.Channels(ctx)
	if err != nil {
		return nil, fmt.Errorf("error reading channels: %w", err)
	}
	if len(channels) == 0 {
		return nil, ErrNoChannels
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
		chstat, err := r.processChannel(ctx, &ch, nil)
		if err != nil {
			return ret, err
		}
		ret.add(chstat)
	}

	return ret, nil
}

func (r *Redownloader) Download(ctx context.Context, cl fileproc.FileGetter) (FileStats, error) {
	var ret FileStats
	channels, err := r.channels(ctx)
	if err != nil {
		return ret, err
	}

	dl := fileproc.NewDownloader(
		ctx,
		true,
		cl,
		fsadapter.NewDirectory(r.src.Name()),
		r.lg,
	)
	defer dl.Stop()

	// determine the file processor for the source.
	fproc, err := r.fileProc(dl)
	if err != nil {
		return ret, err
	}
	defer fproc.Close()

	for _, ch := range channels {
		chstats, err := r.processChannel(ctx, &ch, func(item *dlItem) error {
			if err := fproc.Files(ctx, item.ch, *item.msg, []slack.File{*item.f}); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return ret, err
		}
		ret.add(chstats)
	}
	return ret, nil
}

func (r *Redownloader) processChannel(ctx context.Context, ch *slack.Channel, cb func(item *dlItem) error) (FileStats, error) {
	var ret FileStats

	items, err := r.scanChannel(ctx, ch)
	if err != nil {
		return ret, err
	}
	for _, item := range items {
		if cb != nil {
			if err := cb(&item); err != nil {
				return ret, err
			}
		}
		ret.NumFiles++
		ret.NumBytes += uint64(item.f.Size)
	}
	return ret, nil
}

// dlItem holds the item to be downloaded.
type dlItem struct {
	msg *slack.Message
	f   *slack.File
	ch  *slack.Channel
}

func (r *Redownloader) scanChannel(ctx context.Context, ch *slack.Channel) ([]dlItem, error) {
	r.lg.Info("scanning channel", "channel", ch.ID)
	it, err := r.src.AllMessages(ctx, ch.ID)
	if err != nil {
		if errors.Is(err, source.ErrNotFound) {
			// no data in the channel
			return nil, nil
		}
		return nil, fmt.Errorf("error reading messages: %w", err)
	}

	r.lg.InfoContext(ctx, "scanning messages")
	return r.scanMsgs(ctx, ch, it, false)
}

// scanMsgs scans messages msgs, calling itself recursively for every thread, collecting all files that
// are not present on the file system.
func (r *Redownloader) scanMsgs(ctx context.Context, ch *slack.Channel, msgIt iter.Seq2[slack.Message, error], isThread bool) ([]dlItem, error) {
	ctx, task := trace.NewTask(ctx, "scanMsgs")
	defer task.End()

	trace.Logf(ctx, "scanMsgs", "channel_id=%s, isThread=%v", ch.ID, isThread)

	lg := r.lg.With("channel", ch.ID)
	// workaround for completely missing storage
	pathFn := r.pathFunc()
	var toDl []dlItem
	for m, err := range msgIt {
		if err != nil {
			return nil, err
		}
		if structures.IsThreadStart(&m) && !isThread {
			it, err := r.src.AllThreadMessages(ctx, ch.ID, m.ThreadTimestamp)
			if err != nil {
				return nil, fmt.Errorf("error reading thread messages: %w", err)
			}

			lg.InfoContext(ctx, "scanning thread messages", "thread", m.ThreadTimestamp)
			if res, err := r.scanMsgs(ctx, ch, it, true); err != nil {
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
