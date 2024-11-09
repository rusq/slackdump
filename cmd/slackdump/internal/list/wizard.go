package list

import (
	"context"

	"github.com/charmbracelet/huh"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/cfgui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/dumpui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/updaters"
	"github.com/rusq/slackdump/v3/internal/format"
)

var filename string

func wizUsers(ctx context.Context, _ *base.Command, _ []string) error {
	sess, err := bootstrap.SlackdumpSession(ctx)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return err
	}

	filename = makeFilename("users", sess.Info().TeamID, ".json")
	w := dumpui.Wizard{
		Title:       "List Users",
		Name:        "List",
		LocalConfig: userConfiguration,
		Cmd:         CmdListUsers,
		ArgsFn: func() []string {
			return []string{filename}
		},
	}
	return w.Run(ctx)
}

func wizChannels(ctx context.Context, _ *base.Command, _ []string) error {
	sess, err := bootstrap.SlackdumpSession(ctx)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return err
	}

	filename = makeFilename("channels", sess.Info().TeamID, ".json")
	w := dumpui.Wizard{
		Title:       "List Channels",
		Name:        "List",
		LocalConfig: chanFlags.configuration,
		Cmd:         CmdListChannels,
		ArgsFn: func() []string {
			return []string{filename}
		},
	}
	return w.Run(ctx)
}

func userConfiguration() cfgui.Configuration {
	c := cfgui.Configuration{
		cfgui.ParamGroup{
			Name: "User List Options",
			Params: []cfgui.Parameter{
				filenameParam("users.json"),
			},
		},
	}
	return append(c, commonParams.configuration()...)
}

func filenameParam(placeholder string) cfgui.Parameter {
	return cfgui.Parameter{
		Name:        "Output Filename",
		Value:       filename,
		Description: "The filename to save the output to",
		Inline:      true,
		Updater:     updaters.NewFileNew(&filename, placeholder, true, true),
	}
}

func (o *channelOptions) configuration() cfgui.Configuration {
	c := cfgui.Configuration{
		cfgui.ParamGroup{
			Name: "Channel Options",
			Params: []cfgui.Parameter{
				filenameParam("channels.json"),
				{
					Name:        "Do not Resolve Users",
					Value:       cfgui.Checkbox(o.noResolve),
					Description: "Do not resolve user IDs to names",
					Updater:     updaters.NewBool(&o.noResolve),
				},
			},
		},
		cfgui.ParamGroup{
			Name: "Cache Options",
			Params: []cfgui.Parameter{
				{
					Name:        "Disable Cache",
					Value:       cfgui.Checkbox(o.cache.Disabled),
					Description: "Disable channel cache",
					Updater:     updaters.NewBool(&o.cache.Disabled),
				},
				{
					Name:        "Cache Retention",
					Value:       o.cache.Retention.String(),
					Description: "Channel cache retention time. After this time, the cache is considered stale and will be refreshed.",
					Inline:      true,
					Updater:     updaters.NewDuration(&o.cache.Retention, true),
				},
				{
					Name:        "Cache Filename",
					Value:       o.cache.Filename,
					Description: "The filename of the cache",
					Inline:      true,
					Updater:     updaters.NewString(&o.cache.Filename, "channels.json", true, huh.ValidateNotEmpty()),
				},
			},
		},
	}
	return append(c, commonParams.configuration()...)
}

func (l *listOptions) configuration() cfgui.Configuration {
	c := cfgui.Configuration{
		cfgui.ParamGroup{
			Name: "Common Options",
			Params: []cfgui.Parameter{
				{
					Name:        "List Type",
					Value:       l.listType.String(),
					Description: "The output list type",
					Updater: updaters.NewPicklist(&l.listType, huh.NewSelect[format.Type]().
						Title("List Type").
						Options(
							huh.NewOption("Text", format.CText),
							huh.NewOption("JSON", format.CJSON),
							huh.NewOption("CSV", format.CCSV),
						)),
				},
				{
					Name:        "Quiet Mode",
					Value:       cfgui.Checkbox(l.quiet),
					Description: "Don't print anything on the screen, just save the file",
					Updater:     updaters.NewBool(&l.quiet),
				},
				{
					Name:        "Display Only",
					Value:       cfgui.Checkbox(l.nosave),
					Description: "Don't save the data to a file, just print it to the screen",
					Updater:     updaters.NewBool(&l.nosave),
				},
			},
		},
	}
	return c
}
