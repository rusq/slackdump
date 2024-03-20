package diag

import (
	"context"
	"encoding/json"
	"os"

	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/edge"
	"github.com/rusq/slackdump/v3/logger"
)

var CmdEdge = &base.Command{
	Run:         runEdge,
	Wizard:      func(ctx context.Context, cmd *base.Command, args []string) error { panic("not implemented") },
	UsageLine:   "slack tools edge",
	Short:       "Edge test",
	RequireAuth: true,
	Long: `
# Slack Edge API test tool

Edge test attempts to call the Edge API with the provided credentials.
`,
}

var edgeParams = struct {
	channel string
}{}

func init() {
	CmdEdge.Flag.StringVar(&edgeParams.channel, "channel", "CHY5HUESG", "channel to get users from")
}

func runEdge(ctx context.Context, cmd *base.Command, args []string) error {
	lg := logger.FromContext(ctx)

	prov, err := auth.FromContext(ctx)
	if err != nil {
		base.SetExitStatus(base.SAuthError)
		return err
	}
	ai, err := prov.Test(ctx)
	if err != nil {
		return err
	}
	lg.Printf("auth test: %+v", ai)

	lg.Print("connected")
	sd, err := slackdump.New(ctx, prov)
	if err != nil {
		base.SetExitStatus(base.SAuthError)
		return err
	}

	cl, err := edge.NewWithProvider(cfg.Workspace, sd.Info().TeamID, prov)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}

	lg.Printf("*** Search for Channels test ***")
	channels, err := cl.SearchChannels(ctx, "")
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(channels)

	lg.Printf("*** DMs test ***")
	dms, err := cl.DMs(ctx)
	if err != nil {
		return err
	}
	enc.Encode(dms)

	return nil
}
