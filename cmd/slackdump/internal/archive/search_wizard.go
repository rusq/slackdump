package archive

import (
	"context"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/dumpui"
)

func SearchWizard(ctx context.Context, cmd *base.Command, args []string) error {
	w := &dumpui.Wizard{
		Title: "Dump Search Results",
		Name:  "Search",
		Cmd:   cmdSearchAll,
	}

	return w.Run(ctx)
}
