package emoji

import (
	"context"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/cfgui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/dumpui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/updaters"
)

func wizard(ctx context.Context, cmd *base.Command, args []string) error {
	w := dumpui.Wizard{
		Title:       "Emoji dump",
		Name:        "Emoji",
		Cmd:         CmdEmoji,
		LocalConfig: cmdFlags.configuration,
	}
	return w.Run(ctx)
}

func (o *options) configuration() cfgui.Configuration {
	return cfgui.Configuration{
		cfgui.ParamGroup{
			Name: "API Options",
			Params: []cfgui.Parameter{
				{
					Name:        "Full Emoji Information",
					Value:       cfgui.Checkbox(o.fullInfo),
					Description: "Uses edge API to fetch full emoji information, including usernames",
					Updater:     updaters.NewBool(&o.fullInfo),
				},
			},
		},
		cfgui.ParamGroup{
			Name: "Download Options",
			Params: []cfgui.Parameter{
				{
					Name:        "Ignore Download Errors",
					Value:       cfgui.Checkbox(o.ignoreErrors),
					Description: "Ignore download errors and continue",
					Updater:     updaters.NewBool(&o.ignoreErrors),
				},
			},
		},
	}
}
