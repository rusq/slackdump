package archive

import (
	"context"
	"errors"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/cfgui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/dumpui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/updaters"
)

func wizSearch(ctx context.Context, cmd *base.Command, args []string) error {
	w := &dumpui.Wizard{
		Title:       "Archive Search Results",
		Name:        "Search",
		Cmd:         adapterCmd,
		LocalConfig: searchCfg,
		ArgsFn: func() []string {
			return append([]string{action}, strings.Split(terms, " ")...)
		},
		ValidateParamsFn: func() error {
			if action == "" {
				return errors.New("select action")
			}
			if len(terms) == 0 {
				return errors.New("specify search terms in Search Options")
			}
			return nil
		},
	}
	return w.Run(ctx)
}

const (
	acMessages = "messages"
	acFiles    = "files"
	acAll      = "all"
)

var (
	action string = acMessages
	terms  string
)

func searchCfg() cfgui.Configuration {
	return cfgui.Configuration{
		cfgui.ParamGroup{
			Name: "Required",
			Params: []cfgui.Parameter{
				{
					Name:        "Search Terms",
					Description: "Enter your search query.",
					Value:       terms,
					Inline:      true,
					Updater:     updaters.NewString(&terms, "Search...", false, huh.ValidateNotEmpty()),
				},
			},
		},
		cfgui.ParamGroup{
			Name: "Other parameters",
			Params: []cfgui.Parameter{
				{
					Name:        "Scope",
					Description: "Choose the search scope.",
					Value:       action,
					Inline:      false,
					Updater: updaters.NewPicklist(&action, huh.NewSelect[string]().Options(
						huh.NewOption("messages", acMessages),
						huh.NewOption("files", acFiles),
						huh.NewOption("all", acAll),
					).DescriptionFunc(func() string {
						switch action {
						case acMessages:
							return "Search only in messages"
						case acFiles:
							return "Search only in files"
						case acAll:
							return "Search in both messages and in files"
						default:
							return "undefined search action"
						}
					}, &action).Title("Search Scope")),
				},
			},
		},
	}
}

var adapterCmd = &base.Command{
	Run: adaptercmd,
}

func adaptercmd(ctx context.Context, _ *base.Command, args []string) error {
	if len(args) < 1 {
		panic("internal error: empty arguments")
	}
	if len(args) < 2 {
		return errors.New("no search terms")
	}
	aCmd := args[0]
	var cmd *base.Command
	switch aCmd {
	case acMessages:
		cmd = cmdSearchMessages
	case acFiles:
		cmd = cmdSearchFiles
	case acAll:
		cmd = cmdSearchAll
	default:
		return errors.New("invalid adapter command")
	}

	return cmd.Run(ctx, cmd, args[1:])
}
