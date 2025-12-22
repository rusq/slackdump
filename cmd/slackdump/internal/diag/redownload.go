package diag

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

var cmdRedownload = &base.Command{
	UsageLine: "tools redownload [flags] <archive_dir>",
	Short:     "attempts to redownload missing files from the archive",
	Long: `# File redownload tool
Redownload tool scans the slackdump export, archive or dump directory,
validating the files.

If a file is missing or has zero length, it will be redownloaded from the Slack
API. The tool will not overwrite existing files, so it is safe to run it
multiple times.

**Please note:**

1. It requires you to have a valid authentication in the selected workspace.
2. Ensure that you have selected the correct workspace using "slackdump workspace select".
3. It only support directories.  ZIP files can not be updated. Unpack ZIP file
   to a directory before using this tool.
`,
	FlagMask:    cfg.OmitAll &^ (cfg.OmitAuthFlags | cfg.OmitWorkspaceFlag),
	Run:         runRedownload,
	PrintFlags:  true,
	RequireAuth: true,
}

func runRedownload(ctx context.Context, _ *base.Command, args []string) error {
	if len(args) != 1 {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("expected exactly one argument")
	}
	dir := args[0]

	if err := validate(dir); err != nil {
		return err
	}

	n, err := redownload(ctx, dir)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	if n == 0 {
		slog.Info("no missing files found")
	} else {
		slog.Info("redownloaded missing files", "num_files", n)
	}

	return nil
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

func redownload(ctx context.Context, dir string) (int, error) {
	src, err := source.Load(ctx, dir)
	if err != nil {
		return 0, fmt.Errorf("error opening directory: %w", err)
	}
	defer src.Close()

	channels, err := src.Channels(ctx)
	if err != nil {
		return 0, fmt.Errorf("error reading channels: %w", err)
	}
	if len(channels) == 0 {
		return 0, errors.New("no channels found")
	}
	slog.Info("directory opened", "num_channels", len(channels))

	client, err := bootstrap.Slack(ctx)
	if err != nil {
		return 0, fmt.Errorf("error creating slackdump session: %w", err)
	}
	dl := fileproc.NewDownloader(
		ctx,
		true,
		client,
		fsadapter.NewDirectory(src.Name()),
		cfg.Log,
	)
	defer dl.Stop()

	// determine the file processor for the source.
	fproc, err := fileProcessorForSource(src, dl)
	if err != nil {
		return 0, err
	}
	defer fproc.Close()

	total := 0
	for _, ch := range channels {
		if n, err := redownloadChannel(ctx, fproc, src, &ch); err != nil {
			return total, err
		} else {
			total += n
		}
	}

	return total, nil
}

// fileProcessorForSource returns the appropriate file processor for the given
// source.
func fileProcessorForSource(src source.Sourcer, dl fileproc.Downloader) (processor.Filer, error) {
	var fproc processor.Filer
	srcFlags := src.Type()
	switch {
	case srcFlags&source.FDatabase != 0 || srcFlags&source.FChunk != 0:
		fproc = fileproc.New(dl)
	case srcFlags&source.FExport != 0:
		typ := src.Files().Type()
		if typ == source.STnone {
			typ = source.STmattermost // default to mattermost
		}
		fproc = fileproc.NewExport(typ, dl)
	case srcFlags&source.FDump != 0:
		fproc = fileproc.NewDump(dl)
	default:
		return nil, fmt.Errorf("unable to determine file storage format for the source with flags %s", srcFlags)
	}
	return fproc, nil
}

func redownloadChannel(ctx context.Context, fp processor.Filer, src source.Sourcer, ch *slack.Channel) (int, error) {
	slog.Info("processing channel", "channel", ch.ID)
	it, err := src.AllMessages(ctx, ch.ID)
	if err != nil {
		if errors.Is(err, source.ErrNotFound) {
			// no data in the channel
			return 0, nil
		}
		return 0, fmt.Errorf("error reading messages: %w", err)
	}
	// collect messages from the iterator
	msgs, err := collect(it)
	if err != nil {
		return 0, fmt.Errorf("error fetching messages: %w", err)
	}

	if len(msgs) == 0 {
		return 0, nil
	}
	slog.Info("scanning messages", "num_messages", len(msgs))
	return scanMsgs(ctx, fp, src, ch, msgs, false)
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

func pathFuncForSource(src source.Sourcer) func(ch *slack.Channel, f *slack.File) string {
	if src.Files().Type() != source.STnone {
		// easy
		return src.Files().FilePath
	}
	typ := src.Type()
	switch {
	case typ&source.FDump != 0:
		return source.DumpFilepath
	default:
		// in all other cases we default to mattermost file path.
		return source.MattermostFilepath
	}
	// unreachable
}

func scanMsgs(ctx context.Context, fp processor.Filer, src source.Sourcer, ch *slack.Channel, msgs []slack.Message, isThread bool) (int, error) {
	lg := slog.With("channel", ch.ID)
	// workaround for completely missing storage
	pathFn := pathFuncForSource(src)
	total := 0
	for _, m := range msgs {
		if structures.IsThreadStart(&m) && !isThread {
			it, err := src.AllThreadMessages(ctx, ch.ID, m.ThreadTimestamp)
			if err != nil {
				return 0, fmt.Errorf("error reading thread messages: %w", err)
			}
			tm, err := collect(it)
			if err != nil {
				return 0, fmt.Errorf("error collecting thread messages: %w", err)
			}

			lg.Info("scanning thread messages", "num_messages", len(tm), "thread", m.ThreadTimestamp)
			if n, err := scanMsgs(ctx, fp, src, ch, tm, true); err != nil {
				return total, err
			} else {
				total += n
			}
		}

		// collect all missing files from the message.
		var missing []slack.File
		for _, ff := range m.Files {
			name := filepath.Join(src.Name(), pathFn(ch, &ff))
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
					return total, fmt.Errorf("error accessing file: %w", err)
				}
			} else if fi.Size() == 0 {
				// zero length files are considered missing
				lg.Debug("zero length file")
				missing = append(missing, ff)
			} else {
				lg.Debug("file OK")
			}
		}
		if len(missing) > 0 {
			total += len(missing)
			lg.Info("found missing files", "num_files", len(missing))
			if err := fp.Files(ctx, ch, m, missing); err != nil {
				return total, fmt.Errorf("error processing files: %w", err)
			}
		}
	}
	return total, nil
}
