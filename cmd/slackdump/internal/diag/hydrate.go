package diag

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/rusq/slackdump/v3/internal/chunk/transform/fileproc"
	"github.com/rusq/slackdump/v3/internal/source"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/downloader"
	"github.com/rusq/slackdump/v3/internal/structures"
)

var cmdHydrate = &base.Command{
	UsageLine: "slackdump tools hydrate [flags] <slack_export.zip>",
	Short:     "hydrate slack export with files",
	Long: `
# Hydrate tool

Hydrate command operates on a native Slack Export archive and downloads all files
from messages and posts into a __uploads directory in the archive.

It creates a copy of the original archive and adds file to it, so you will need
at least as much free space as the original archive occupies + the size of all files
to be downloaded.  It also uses a temporary directory to store the files before
repacking the archive.

If the old file was named "my_export.zip", the new file will be named
"my_export-hydrated.zip".  Optionally you can specify the name of the output file
with the -o flag.

__Please note__:  this command is only compatible with native Slack Exports that
have an active export token which is valid and not expired or revoked.
`,
	Run:        runHydrate,
	FlagMask:   cfg.OmitAll,
	PrintFlags: true,
}

var (
	outfile string
	dryrun  bool
)

func init() {
	cmdHydrate.Flag.StringVar(&outfile, "o", "", "output file name")
	cmdHydrate.Flag.BoolVar(&dryrun, "dry-run", false, "do not download files, just print what would be done")
}

func runHydrate(ctx context.Context, cmd *base.Command, args []string) error {
	lg := cfg.Log
	if len(args) != 1 {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("expected exactly one argument")
	}
	archive := args[0]
	if filepath.Ext(strings.ToLower(archive)) != ".zip" {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("expected a zip file")
	}
	if outfile == "" {
		outfile = archive[:len(archive)-4] + "-hydrated.zip"
	}
	lg = lg.With("input", archive, "output", outfile)

	tmpdir, err := os.MkdirTemp("", "slackdump-hydrate")
	if err != nil {
		base.SetExitStatus(base.SGenericError)
		return fmt.Errorf("error creating temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpdir)
	lg.InfoContext(ctx, "using temporary directory", "name", tmpdir)

	lg.InfoContext(ctx, "downloading files")
	if err := download(ctx, archive, tmpdir, dryrun); err != nil {
		base.SetExitStatus(base.SGenericError)
		return fmt.Errorf("error downloading files: %w", err)
	}
	if dryrun {
		lg.InfoContext(ctx, "dry-run mode, not repacking the archive")
		return nil
	}

	lg.InfoContext(ctx, "repacking the archive")
	if err := extract(tmpdir, archive); err != nil {
		base.SetExitStatus(base.SGenericError)
		return fmt.Errorf("error extracting the archive: %w", err)
	}
	if err := packDir(outfile, tmpdir); err != nil {
		base.SetExitStatus(base.SGenericError)
		return fmt.Errorf("error creating the new archive: %w", err)
	}

	return nil
}

func extract(target, archive string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return err
	}

	zr, err := zip.NewReader(f, fi.Size())
	if err != nil {
		return err
	}
	if err := os.CopyFS(target, zr); err != nil {
		return err
	}

	return nil
}

func packDir(zipfile string, dir string) error {
	fsys := os.DirFS(dir)
	f, err := os.Create(zipfile)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	return zw.AddFS(fsys)
}

// download downloads all files from messages and posts into a __uploads directory in the archive.
func download(ctx context.Context, archive, target string, dry bool) error {
	// Open the archive
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer zr.Close()

	src, err := source.NewExport(zr, archive)
	if err != nil {
		return err
	}
	defer src.Close()

	trg := fsadapter.NewDirectory(target)
	defer trg.Close()

	var d downloader.GetFiler
	if dry {
		d = debugdl{}
	} else {
		d = httpget{}
	}

	if err := downloadFiles(ctx, d, trg, src); err != nil {
		return err
	}

	return nil
}

//go:generate mockgen -destination=hydrate_mock_test.go -package=diag -source hydrate.go sourcer
type sourcer interface {
	Channels(ctx context.Context) ([]slack.Channel, error)
	AllMessages(ctx context.Context, channelID string) (iter.Seq2[slack.Message, error], error)
	AllThreadMessages(ctx context.Context, channelID, threadTimestamp string) (iter.Seq2[slack.Message, error], error)
}

func downloadFiles(ctx context.Context, d downloader.GetFiler, trg fsadapter.FS, src sourcer) error {
	dl := downloader.New(d, trg, downloader.WithLogger(cfg.Log))
	if err := dl.Start(ctx); err != nil {
		return err
	}
	defer dl.Stop()

	proc := fileproc.New(dl)

	channels, err := src.Channels(ctx)
	if err != nil {
		return fmt.Errorf("error reading channels: %w", err)
	}

	for _, ch := range channels {
		msgs, err := src.AllMessages(ctx, ch.ID)
		if err != nil {
			return fmt.Errorf("error reading messages in channel %s: %w", ch.ID, err)
		}
		for m, err := range msgs {
			if err != nil {
				return fmt.Errorf("error reading message in channel %s: %w", ch.ID, err)
			}
			if len(m.Files) > 0 {
				if err := proc.Files(ctx, &ch, m, m.Files); err != nil {
					return fmt.Errorf("error processing files in message %s: %w", m.Timestamp, err)
				}
			}
			if structures.IsThreadStart(&m) {
				itTm, err := src.AllThreadMessages(ctx, ch.ID, m.ThreadTimestamp)
				if err != nil {
					return fmt.Errorf("error reading thread messages for message %s in channel %s: %w", m.Timestamp, ch.ID, err)
				}
				for tm, err := range itTm {
					if err != nil {
						return fmt.Errorf("error reading thread message %s in channel %s: %w", tm.Timestamp, ch.ID, err)
					}
					if len(tm.Files) > 0 {
						if err := proc.Files(ctx, &ch, tm, tm.Files); err != nil {
							return fmt.Errorf("error processing files in thread message %s: %w", tm.Timestamp, err)
						}
					}
				}
			}
		}
	}

	return nil
}

// httpget is an implementation of downloader that should be sufficient to download
// files from the official export, as they have the token included.
type httpget struct{}

var (
	errUrlParse = errors.New("invalid URL")
	errNoToken  = errors.New("missing token in the URL")
)

func (httpget) GetFileContext(ctx context.Context, downloadURL string, w io.Writer) error {
	u, err := url.Parse(downloadURL)
	if err != nil {
		return fmt.Errorf("%w: %w", errUrlParse, err)
	}
	if u.Query().Get("t") == "" {
		// won't be able to download without the token
		return errNoToken
	}

	// url seems valid
	resp, err := http.Get(downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %s", resp.Status)
	}
	_, err = io.Copy(w, resp.Body)
	return err
}

type debugdl struct{}

func (debugdl) GetFileContext(ctx context.Context, downloadURL string, w io.Writer) error {
	slog.Info("would download", "url", downloadURL)
	return nil
}
