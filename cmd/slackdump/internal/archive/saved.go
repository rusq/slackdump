package archive

import (
	"context"
	"encoding/json"
	"os"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/edge"
)

var cmdSaved = &base.Command{
	UsageLine:   "slackdump saved [flags]",
	Short:       "archive saved messages",
	CustomFlags: false,
	FlagMask:    0, // TODO
	PrintFlags:  true,
	RequireAuth: true,
	Run:         runSavedList,
}

var listSavedFlags = struct {
	Filter edge.SavedListFilter
}{
	Filter: edge.SavedListFilterSaved,
}

func init() {
	cmdSaved.Flag.StringVar((*string)(&listSavedFlags.Filter), "filter", string(listSavedFlags.Filter), "Filter for saved messages. Options: saved,archived,completed")
}

func runSavedList(ctx context.Context, cmd *base.Command, args []string) error {
	prov, err := auth.FromContext(ctx)
	if err != nil {
		base.SetExitStatus(base.SAuthError)
		return err
	}
	cl, err := edge.New(ctx, prov)
	if err != nil {
		base.SetExitStatus(base.SAuthError)
		return err
	}
	defer cl.Close()

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	for resp, err := range cl.SavedList(ctx, listSavedFlags.Filter) {
		if err != nil {
			return err
		}
		if err := enc.Encode(resp); err != nil {
			return err
		}
	}

	return nil
}
