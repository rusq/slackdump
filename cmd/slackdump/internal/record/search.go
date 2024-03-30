package record

import (
	"context"
	"errors"
	"strings"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/control"
)

var CmdSearch = &base.Command{
	UsageLine:   "slackdump search [flags] query terms",
	Short:       "records search results matching the given query",
	Long:        `Searches for messages matching criteria.`,
	RequireAuth: true,
	Run:         runSearch,
	PrintFlags:  true,
}

func runSearch(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) == 0 {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("missing query parameter")
	}
	query := strings.Join(args, " ")

	sess, err := cfg.SlackdumpSession(ctx)
	if err != nil {
		return err
	}

	cd, err := chunk.CreateDir(cfg.Output)
	if err != nil {
		base.SetExitStatus(base.SGenericError)
		return err
	}
	defer cd.Close()

	stream := sess.Stream()
	ctrl := control.NewSearch(cd, stream)

	return ctrl.Search(ctx, query)
}
