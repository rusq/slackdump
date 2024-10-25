package dump

import (
	"context"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/dumpui"
)

func WizDump(ctx context.Context, cmd *base.Command, args []string) error {
	w := dumpui.Wizard{
		Title:       "Dump Slackdump channels",
		Particulars: "Dump",
		Cmd:         cmd,
	}
	return w.Run(ctx)
}
