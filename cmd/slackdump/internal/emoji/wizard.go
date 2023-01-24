package emoji

import (
	"context"
	"fmt"

	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/internal/ui"
)

func wizard(ctx context.Context, cmd *base.Command, args []string) error {
	var baseDir string
	for {
		var err error
		baseDir, err = ui.FileSelector("Enter directory or ZIP file name: ", "Emojis will be saved to this directory or ZIP file")
		if err != nil {
			return err
		}
		if baseDir != "-" && baseDir != "" {
			break
		}
		fmt.Println("invalid filename")
	}
	cfg.BaseLoc = baseDir

	var err error
	ignoreErrors, err = ui.Confirm("Ignore download errors?", true)
	if err != nil {
		return err
	}
	return run(ctx, cmd, args)
}
