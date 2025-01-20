package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"

	"github.com/rusq/osenv/v2"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/view"
	"github.com/rusq/slackdump/v3/internal/viewer"
)

var CmdServer = &base.Command{
	UsageLine:  "slackdump server [flags] <archive or zip file>",
	Short:      "starts an attachment web server",
	Run:        runServer,
	PrintFlags: true,
	FlagMask:   cfg.OmitAll,
}

var flags struct {
	ngrokAPIKey string
}

func init() {
	CmdServer.Flag.StringVar(&flags.ngrokAPIKey, "ngrok-token", osenv.Secret("NGROK_AUTHTOKEN", ""), "NGROK API key")
}

func runServer(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) != 1 {
		base.SetExitStatus(base.SInvalidParameters)
		cmd.Usage()
		return errors.New("invalid number of arguments")
	}
	if flags.ngrokAPIKey == "" {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("ngrok token required")
	}

	l, err := ngrok.Listen(ctx, config.HTTPEndpoint(), ngrok.WithAuthtoken(flags.ngrokAPIKey))
	if err != nil {
		return err
	}
	defer l.Close()

	src, err := view.LoadSource(ctx, args[0])
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return err
	}

	slog.InfoContext(ctx, "listener initialised", "url", l.URL())
	// update file paths

	// start server
	mux := http.NewServeMux()
	mux.Handle("/healthcheck", http.HandlerFunc(healthcheck))
	mux.Handle("/file/{id}/{filename}", viewer.NewFileHandler(src, cfg.Log))
	if err := http.Serve(l, middleware.Logger(mux)); err != nil {
		if !strings.EqualFold(fmt.Sprintf("%T", err), "ngrok.errAcceptFailed") {
			return err
		}
		slog.InfoContext(ctx, "server closed")
	}
	return nil
}
