package dump

import (
	"context"
	"strings"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/cfgui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/dumpui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/updaters"
	"github.com/rusq/slackdump/v3/internal/nametmpl"
	"github.com/rusq/slackdump/v3/internal/structures"
)

func WizDump(ctx context.Context, cmd *base.Command, args []string) error {
	w := dumpui.Wizard{
		Title:       "Dump Slack Channels",
		Name:        "Dump",
		LocalConfig: opts.configuration,
		Cmd:         cmd,
		ArgsFn: func() []string {
			return splitEntryList(entryList)
		},
		ValidateParamsFn: func() error {
			return structures.ValidateEntityList(entryList)
		},
	}
	return w.Run(ctx)
}

var entryList string

func splitEntryList(s string) []string {
	return strings.Split(s, " ")
}

func (o *options) configuration() cfgui.Configuration {
	return cfgui.Configuration{
		{
			Name: "Required",
			Params: []cfgui.Parameter{
				cfgui.ChannelIDs(&entryList),
			},
		}, {
			Name: "Optional",
			Params: []cfgui.Parameter{
				{
					Name:        "File naming template",
					Value:       o.nameTemplate,
					Description: "Output file naming template",
					Inline:      true,
					Updater: updaters.NewString(&o.nameTemplate, nametmpl.Default, false, func(s string) error {
						_, err := nametmpl.New(s)
						return err
					}),
				},
				{
					Name:        "V2 Compatibility mode",
					Value:       cfgui.Checkbox(o.compat),
					Description: "Use V2 compatibility mode (slower)",
					Updater:     updaters.NewBool(&o.compat),
				},
				{
					Name:        "Update links",
					Value:       cfgui.Checkbox(o.updateLinks),
					Description: "Update file links to point to the downloaded files",
					Updater:     updaters.NewBool(&o.updateLinks),
				},
			},
		},
	}
}
