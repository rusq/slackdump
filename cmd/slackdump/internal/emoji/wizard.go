package emoji

import (
	"context"
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/bubbles/status"
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

type model struct {
	s status.Model
}

func newModel() *model {
	style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder(), true)
	return &model{
		s: status.New(2, style, []status.Parameter{
			{Name: "Output", Value: cfg.Output},
		}),
	}
}
