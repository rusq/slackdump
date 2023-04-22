package export

import (
	"context"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/export"
	"github.com/rusq/slackdump/v2/internal/ui"
	"github.com/rusq/slackdump/v2/internal/ui/ask"
	"github.com/rusq/slackdump/v2/logger"
)

func wizExport(ctx context.Context, cmd *base.Command, args []string) error {
	options.Logger = logger.FromContext(ctx)
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
	fsa, err := fsadapter.New(baseLoc)
	if err != nil {
		return err
	}
	defer fsa.Close()

	sess, err := slackdump.New(ctx, prov, slackdump.WithFilesystem(fsa), slackdump.WithLogger(options.Logger))
	if err != nil {
		return err
	}
	// TODO v3
	exp := export.New(sess, fsa, options)

	// run export
	return exp.Run(ctx)
}
