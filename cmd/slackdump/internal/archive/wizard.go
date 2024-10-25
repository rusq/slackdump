package archive

import (
	"context"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/dumpui"
)

func archiveWizard(ctx context.Context, cmd *base.Command, args []string) error {
	w := &dumpui.Wizard{
		Title:       "Archive Slack Workspace",
		Particulars: "Archive",
		Cmd:         cmd,
	}
	return w.Run(ctx)
}
