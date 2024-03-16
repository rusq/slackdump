package view

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/viewer"
	"github.com/rusq/slackdump/v3/internal/viewer/source"
	"github.com/rusq/slackdump/v3/logger"
)

var CmdView = &base.Command{
	Short:     "View the slackdump files",
	UsageLine: "slackdump view [flags]",
	Long: `
View the slackdump files.
`,
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
	fi, err := os.Stat(args[0])
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return fmt.Errorf("not found: %s", args[0])
	}
	if !fi.IsDir() {
		base.SetExitStatus(base.SUserError)
		return fmt.Errorf("not a directory: %s", args[0])
	}

	dir, err := chunk.OpenDir(args[0])
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return err
	}
	defer dir.Close()

	src := source.NewChunkDir(dir)

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

	lg := logger.FromContext(ctx)

	lg.Printf("listening on %s", listenAddr)
	if err := v.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			lg.Print("bye")
			return nil
		}
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	return nil
}
