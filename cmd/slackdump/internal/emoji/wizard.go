package emoji

import (
	"context"
	"fmt"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/ui"
)

func wizard(ctx context.Context, cmd *base.Command, args []string) error {
	var baseloc string
	for {
		var err error
		baseloc, err = ui.FileSelector("Enter directory or ZIP file name: ", "Emojis will be saved to this directory or ZIP file")
		if err != nil {
			return err
		}
		if baseloc != "-" && baseloc != "" {
			break
		}
		fmt.Println("invalid filename")
	}
	cfg.Output = baseloc

	var err error
	ignoreErrors, err = ui.Confirm("Ignore download errors?", true)
	if err != nil {
		return err
	}
	return run(ctx, cmd, args)
}
