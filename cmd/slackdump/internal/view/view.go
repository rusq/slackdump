package view

import (
	"archive/zip"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"

	br "github.com/pkg/browser"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/viewer"
	"github.com/rusq/slackdump/v3/internal/viewer/source"
)

//go:embed assets/view.md
var mdView string

var CmdView = &base.Command{
	Short:      "View the slackdump files",
	UsageLine:  "slackdump view [flags]",
	Long:       mdView,
	PrintFlags: true,
	FlagMask:   cfg.OmitAll,
	Run:        RunView,
}

var listenAddr string

func init() {
	CmdView.Flag.StringVar(&listenAddr, "listen", "localhost:8080", "address to listen on")
}

func RunView(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) < 1 {
		base.SetExitStatus(base.SInvalidParameters)
		return fmt.Errorf("viewing slackdump files requires at least one argument")
	}
	src, err := loadSource(ctx, args[0])
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return err
	}
	if cl, ok := src.(io.Closer); ok {
		defer cl.Close()
	}

	v, err := viewer.New(ctx, listenAddr, src)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	// sentinel
	go func() {
		<-ctx.Done()
		v.Close()
	}()

	lg := cfg.Log

	lg.InfoContext(ctx, "listening on", "addr", listenAddr)
	go func() {
		if err := br.OpenURL(fmt.Sprintf("http://%s", listenAddr)); err != nil {
			lg.WarnContext(ctx, "unable to open browser", "error", err)
		}
	}()
	if err := v.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			lg.InfoContext(ctx, "bye")
			return nil
		}
		base.SetExitStatus(base.SApplicationError)
		return err
	}

	return nil
}

type sourceFlags int16

const (
	sfUnknown   sourceFlags = 0
	sfDirectory sourceFlags = 1 << iota
	sfZIP
	sfChunk
	sfExport
	sfDump
)

func loadSource(ctx context.Context, src string) (viewer.Sourcer, error) {
	lg := cfg.Log.With("source", src)
	fi, err := os.Stat(src)
	if err != nil {
		return nil, err
	}
	switch srcType(src, fi) {
	case sfChunk | sfDirectory:
		lg.DebugContext(ctx, "loading chunk directory")
		dir, err := chunk.OpenDir(src)
		if err != nil {
			return nil, err
		}
		return source.NewChunkDir(dir), nil
	case sfExport | sfZIP:
		lg.DebugContext(ctx, "loading export zip")
		f, err := zip.OpenReader(src)
		if err != nil {
			return nil, err
		}
		return source.NewExport(f, src)
	case sfExport | sfDirectory:
		lg.DebugContext(ctx, "loading export directory")
		return source.NewExport(os.DirFS(src), src)
	case sfDump | sfZIP:
		lg.DebugContext(ctx, "loading dump zip")
		f, err := zip.OpenReader(src)
		if err != nil {
			return nil, err
		}
		return source.NewDump(f, src)
	case sfDump | sfDirectory:
		lg.DebugContext(ctx, "loading dump directory")
		return source.NewDump(os.DirFS(src), src)
	default:
		return nil, fmt.Errorf("unsupported source type: %s", src)
	}
}

func srcType(src string, fi fs.FileInfo) sourceFlags {
	var fsys fs.FS // this will be our media for accessing files
	var flags sourceFlags
	if fi.IsDir() {
		fsys = os.DirFS(src)
		flags |= sfDirectory
	} else if fi.Mode().IsRegular() && strings.ToLower(path.Ext(src)) == ".zip" {
		f, err := zip.OpenReader(src)
		if err != nil {
			return sfUnknown
		}
		defer f.Close()
		fsys = f
		flags |= sfZIP
	}
	if ff, err := fs.Glob(fsys, "[CD]*.json"); err == nil && len(ff) > 0 {
		return flags | sfDump
	}
	if _, err := fs.Stat(fsys, "workspace.json.gz"); err == nil {
		if flags&sfZIP != 0 {
			return sfUnknown // compressed chunk directories are not supported
		}
		return flags | sfChunk
	}
	if _, err := fs.Stat(fsys, "channels.json"); err == nil {
		return flags | sfExport
	}
	return sfUnknown
}
