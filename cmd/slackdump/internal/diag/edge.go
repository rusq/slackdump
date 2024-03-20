package diag

import (
	"context"
	"encoding/json"
	"fmt"
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

	lg.Print("connected")
	sd, err := slackdump.New(ctx, prov)
	if err != nil {
		base.SetExitStatus(base.SAuthError)
		return err
	}
	cl, err := edge.New(cfg.Workspace, sd.Info().TeamID, prov.SlackToken(), prov.Cookies())
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	req := edge.UsersListRequest{
		Channels: []string{edgeParams.channel},
		Filter:   "everyone AND NOT bots AND NOT apps",
		Count:    20,
	}
	resp, err := cl.Post(ctx, "/users/list", &req)
	if err != nil {
		return err
	}
	var ur edge.UsersListResponse
	if err := cl.ParseResponse(&ur, resp); err != nil {
		return err
	}
	if !ur.Ok {
		return fmt.Errorf("error: %s", ur.Error)
	}
	lg.Printf("%+v\n", ur)

	lg.Printf("*** Search for Channels test ***")
	channels, err := cl.SearchChannels(ctx, "")
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(channels)

	return nil
}
