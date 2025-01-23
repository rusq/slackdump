package view

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"

	"github.com/rusq/slackdump/v3/internal/source"

	br "github.com/pkg/browser"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/viewer"
)

//go:embed assets/view.md
var mdView string

var CmdView = &base.Command{
	Short:      "View the slackdump files",
	UsageLine:  "slackdump view [flags]",
	Long:       mdView,
	PrintFlags: true,
	FlagMask:   cfg.OmitAll,
	Run:        runView,
}

var listenAddr string

func init() {
	CmdView.Flag.StringVar(&listenAddr, "listen", "127.0.0.1:8080", "address to listen on")
}

func runView(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) < 1 {
		base.SetExitStatus(base.SInvalidParameters)
		return fmt.Errorf("viewing slackdump files requires at least one argument")
	}
	src, err := source.Load(ctx, args[0])
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return err
	}
	defer src.Close()

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
