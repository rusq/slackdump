package export

import (
	"context"

	"github.com/rusq/dlog"
	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/export"
	"github.com/rusq/slackdump/v2/internal/ui"
	"github.com/rusq/slackdump/v2/internal/ui/ask"
)

func wizExport(ctx context.Context, cmd *base.Command, args []string) error {
	options.Logger = dlog.FromContext(ctx)
	prov, err := auth.FromContext(ctx)
	if err != nil {
		return err
	}
	// ask for the list
	list, err := ask.ConversationList("Enter conversations to export (optional)?")
	if err != nil {
		return err
	}
	options.List = list

	// ask if user wants time range
	options.Oldest, options.Latest, err = ask.MaybeTimeRange()
	if err != nil {
		return err
	}

	// ask for the type
	exportType, err := ask.ExportType()
	if err != nil {
		return err
	} else {
		options.Type = exportType
	}

	if wantExportToken, err := ui.Confirm("Do you want to specify an export token for attachments?", false); err != nil {
		return err
	} else if wantExportToken {
		// ask for the export token
		exportToken, err := ui.String("Export token", "Enter the export token, that will be appended to each of the attachment URLs.")
		if err != nil {
			return err
		}
		options.ExportToken = exportToken
	}

	// ask for the save location
	baseLoc, err := ui.FileSelector("Output ZIP or Directory name", "Enter the name of the ZIP or directory to save the export to.")
	if err != nil {
		return err
	}
	cfg.BaseLocation = baseLoc

	sess, err := slackdump.New(ctx, prov, cfg.SlackConfig)
	if err != nil {
		return err
	}
	defer sess.Close()

	exp := export.New(sess, options)

	// run export
	return exp.Run(ctx)
}
