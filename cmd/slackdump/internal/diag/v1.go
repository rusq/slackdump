package diag

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/diag/sdv1"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/backend/dbase"
)

var cmdV1 = &base.Command{
	UsageLine: "slackdump v1 [flags] <path>",
	Short:     "slackdump v1.0.x conversion utility",
	Long: `# Conversion utility for slackdump v1.0.x files

Slackdump v1.0.x are rare in the wild, but if you have one, you can use this
command to convert it to current dump format to be able to use it with the
viewer and other commands.`,
	CustomFlags: false,
	FlagMask:    cfg.OmitAll &^ cfg.OmitOutputFlag,
	PrintFlags:  true,
	RequireAuth: false,
	HideWizard:  true,
	Run:         runV1,
}

func runV1(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("must provide a single path to v1.0.x dump")
	}
	path := args[0]

	output := cfg.StripZipExt(cfg.Output)

	if err := os.MkdirAll(output, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	wconn, si, err := bootstrap.Database(output, "v1")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer wconn.Close()

	if err := fs.WalkDir(os.DirFS(path), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk directory: %w", err)
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if match, err := filepath.Match("[CDG]*.json", name); err != nil || !match {
			return nil
		}
		erc, err := dbase.New(ctx, wconn, si, dbase.WithVerbose(cfg.Verbose))
		if err != nil {
			return fmt.Errorf("failed to create new session: %w", err)
		}
		defer erc.Close()
		if err := convertOne(ctx, erc, filepath.Join(path, p), output); err != nil {
			return fmt.Errorf("failed to convert file %q: %w", p, err)
		}
		return nil
	}); err != nil {
		return nil
	}

	slog.InfoContext(ctx, "v1.0.x dump converted successfully", "output", output)
	return nil
}

func convertOne(ctx context.Context, erc chunk.Encoder, path string, outputDir string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file %q: %w", path, err)
	}
	v1, err := sdv1.Load(path)
	if err != nil {
		return fmt.Errorf("failed to load v1.0.x dump: %w", err)
	}
	mm := v1.Msgs()
	if err := erc.Encode(ctx, &chunk.Chunk{
		ChannelID: v1.ChannelID,
		Type:      chunk.CMessages,
		Timestamp: fi.ModTime().UnixMicro(),
		Count:     int32(len(mm)),
		IsLast:    true,
		Messages:  v1.Msgs(),
	}); err != nil {
		return fmt.Errorf("failed to encode chunk: %w", err)
	}
	if err := erc.Encode(ctx, &chunk.Chunk{
		Type:      chunk.CUsers,
		Timestamp: fi.ModTime().UnixMicro(),
		Count:     int32(len(v1.SD.Users.Users)),
		Users:     v1.SD.Users.Users,
	}); err != nil {
		return fmt.Errorf("failed to encode users: %w", err)
	}
	if err := erc.Encode(ctx, &chunk.Chunk{
		Type:      chunk.CChannels,
		Timestamp: fi.ModTime().UnixMicro(),
		Count:     int32(len(v1.SD.Channels)),
		Channels:  v1.SD.Channels,
	}); err != nil {
		return fmt.Errorf("failed to encode channels: %w", err)
	}
	// find this conversation and insert it as a channel info
	if err := erc.Encode(ctx, &chunk.Chunk{
		Type:      chunk.CChannelInfo,
		Timestamp: fi.ModTime().UnixMicro(),
		ChannelID: v1.ChannelID,
		Channel:   v1.ChannelInfo(),
	}); err != nil {
		return fmt.Errorf("failed to encode channel info: %w", err)
	}

	srcdir := filepath.Dir(path)
	attachDir := filepath.Join(srcdir, v1.ChannelID)
	if fi, err := os.Stat(attachDir); err == nil && fi.IsDir() {
		// there are attachments.
		srcfs := os.DirFS(attachDir)
		if err := os.MkdirAll(filepath.Join(path, v1.ChannelID), 0o755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
		if err := os.CopyFS(path, srcfs); err != nil {
			return fmt.Errorf("failed to copy attachments: %w", err)
		}
	}

	return nil
}

func encode(filename string, a any) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w", filename, err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")

	if err := enc.Encode(a); err != nil {
		return fmt.Errorf("failed to encode file %q: %w", filename, err)
	}
	return nil
}
