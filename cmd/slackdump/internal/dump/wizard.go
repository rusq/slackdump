package dump

import (
	"context"
	"errors"
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
			return validateEntryList(entryList)
		},
	}
	return w.Run(ctx)
}

var entryList string

func splitEntryList(s string) []string {
	return strings.Split(s, " ")
}

func validateEntryList(s string) error {
	if len(s) == 0 {
		return errors.New("no entries")
	}
	ee := strings.Split(s, " ")
	if len(ee) == 0 {
		return errors.New("no entries")
	}
	_, err := structures.NewEntityList(ee)
	if err != nil {
		return err
	}
	return nil
}

func (o *options) configuration() cfgui.Configuration {
	return cfgui.Configuration{
		{
			Name: "Input",
			Params: []cfgui.Parameter{
				{
					Name:        "Channel IDs or URLs",
					Value:       entryList,
					Description: "List of channel IDs or URLs to dump",
					Inline:      true,
					Updater:     updaters.NewString(&entryList, "", true, validateEntryList),
				},
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
					Name:        "Compatibility mode",
					Value:       cfgui.Checkbox(o.compat),
					Description: "Use compatibility mode",
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
