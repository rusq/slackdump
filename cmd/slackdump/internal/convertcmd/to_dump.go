package convertcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/transform"
	"github.com/rusq/slackdump/v3/internal/convert"
	"github.com/rusq/slackdump/v3/internal/source"
	"github.com/rusq/slackdump/v3/internal/structures"
)

var ErrMeaningless = errors.New("meaningless conversion")

func toDump(ctx context.Context, srcpath, trgloc string, cflg convertflags) error {
	st, err := source.Type(srcpath)
	if err != nil {
		return err
	}
	if st.Has(source.FDump) {
		return ErrMeaningless
	}
	src, err := source.Load(ctx, srcpath)
	if err != nil {
		return err
	}
	defer src.Close()

	fsa, err := fsadapter.New(trgloc)
	if err != nil {
		return err
	}
	defer fsa.Close()

	filesEnabled := cflg.includeFiles && (st.Has(source.FMattermost))

	fh := &fileHandler{
		fc: convert.NewFileCopier(src, fsa, src.Files().FilePath, source.DumpFilepath, filesEnabled),
	}

	dw := NewDumpWriter(fsa)
	conv, err := transform.NewDumpConverter(
		fsa,
		src,
		transform.DumpWithLogger(cfg.Log),
		transform.DumpWithPipeline(fh.copyFiles),
	)
	if err != nil {
		return err
	}

	channels, err := src.Channels(ctx)
	if err != nil {
		return err
	}
	if err := dw.Channels(channels); err != nil {
		return err
	}

	for _, c := range channels {
		if err := conv.Convert(ctx, chunk.ToFileID(c.ID, "", false)); err != nil {
			return err
		}
	}
	users, err := src.Users(ctx)
	if err != nil {
		return err
	}
	if err := dw.Users(users); err != nil {
		return err
	}

	wi, err := src.WorkspaceInfo(ctx)
	if err != nil {
		return err
	}
	if err := dw.WorkspaceInfo(wi); err != nil {
		return err
	}

	return nil
}

type fileHandler struct {
	fc *convert.FileCopier
}

// copyFiles is a pipeline function that extracts files from messages and
// calls the file copier.
func (f *fileHandler) copyFiles(channelID string, _ string, mm []slack.Message) error {
	for _, m := range mm {
		if err := f.fc.Copy(structures.ChannelFromID(channelID), &m); err != nil {
			return err
		}
	}
	return nil
}

type DumpWriter struct {
	fsa fsadapter.FS
}

func NewDumpWriter(fsa fsadapter.FS) *DumpWriter {
	return &DumpWriter{fsa: fsa}
}

func (dw *DumpWriter) Users(uu []slack.User) error {
	return marshalFormatted(dw.fsa, "users.json", uu)
}

func (dw *DumpWriter) Channels(cc []slack.Channel) error {
	return marshalFormatted(dw.fsa, "channels.json", cc)
}

func (dw *DumpWriter) WorkspaceInfo(wi *slack.AuthTestResponse) error {
	return marshalFormatted(dw.fsa, "workspace.json", wi)
}

func marshalFormatted(fsa fsadapter.FS, filename string, a any) error {
	f, err := fsa.Create(filename)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", " ")
	if err := enc.Encode(a); err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	return nil
}
