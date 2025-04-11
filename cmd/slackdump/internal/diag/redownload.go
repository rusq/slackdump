package diag

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/convert/transform/fileproc"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/processor"
	"github.com/rusq/slackdump/v3/source"
)

var cmdRedownload = &base.Command{
	UsageLine: "tools redownload [flags] <archive_dir>",
	Short:     "attempts to redownload missing files from the archive",
	Long: `# File redownload tool
Redownload tool scans the directory with Slackdump Archive, validating the files.
If a file is missing or has zero length, it will be redownloaded from the Slack API.
The tool will not overwrite existing files, so it is safe to run multiple times.

Please note:

1. It requires you to have a valid authentication in the selected workspace.
2. Ensure that you have selected the correct workspace using "slackdump workspace select".
3. It only works with Slackdump Archive directories, Slack exports and dumps
are not supported.`,
	FlagMask:    cfg.OmitAll &^ cfg.OmitAuthFlags,
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
	if flags&source.FChunk == 0 {
		base.SetExitStatus(base.SUserError)
		return errors.New("expected a Slackdump Archive directory")
	}

	if fi, err := os.Stat(filepath.Join(dir, string(chunk.FWorkspace)+".json.gz")); err != nil {
		base.SetExitStatus(base.SUserError)
		return fmt.Errorf("error accessing the workspace file: %w", err)
	} else if fi.IsDir() {
		base.SetExitStatus(base.SUserError)
		return errors.New("this does not look like an archive directory")
	}
	return nil
}

func redownload(ctx context.Context, dir string) (int, error) {
	cd, err := chunk.OpenDir(dir)
	if err != nil {
		return 0, fmt.Errorf("error opening directory: %w", err)
	}
	defer cd.Close()

	channels, err := cd.Channels(ctx)
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
		fsadapter.NewDirectory(cd.Name()),
		cfg.Log,
	)
	defer dl.Stop()
	// we are using the same file subprocessor as the mattermost export.
	fproc := fileproc.New(dl)

	total := 0
	for _, ch := range channels {
		if n, err := redlChannel(ctx, fproc, cd, &ch); err != nil {
			return total, err
		} else {
			total += n
		}
	}

	return total, nil
}

func redlChannel(ctx context.Context, fp processor.Filer, cd *chunk.Directory, ch *slack.Channel) (int, error) {
	slog.Info("processing channel", "channel", ch.ID)
	f, err := cd.Open(chunk.FileID(ch.ID))
	if err != nil {
		return 0, fmt.Errorf("error reading messages: %w", err)
	}
	defer f.Close()
	msgs, err := f.AllMessages(ctx, ch.ID)
	if err != nil {
		return 0, fmt.Errorf("error reading messages: %w", err)
	}
	if len(msgs) == 0 {
		return 0, nil
	}
	slog.Info("scanning messages", "num_messages", len(msgs))
	return scanMsgs(ctx, fp, cd, f, ch, msgs)
}

func scanMsgs(ctx context.Context, fp processor.Filer, cd *chunk.Directory, f *chunk.File, ch *slack.Channel, msgs []slack.Message) (int, error) {
	lg := slog.With("channel", ch.ID)
	total := 0
	for _, m := range msgs {
		if structures.IsThreadStart(&m) {
			tm, err := f.AllThreadMessages(ch.ID, m.ThreadTimestamp)
			if err != nil {
				return 0, fmt.Errorf("error reading thread messages: %w", err)
			}
			lg.Info("scanning thread messages", "num_messages", len(tm), "thread", m.ThreadTimestamp)
			if n, err := scanMsgs(ctx, fp, cd, f, ch, tm); err != nil {
				return total, err
			} else {
				total += n
			}
		}

		// collect all missing files from the message.
		var missing []slack.File
		for _, ff := range m.Files {
			name := filepath.Join(cd.Name(), source.MattermostFilepath(ch, &ff))
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
