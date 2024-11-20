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

	filename = ""
	w := dumpui.Wizard{
		Title:       "List Users",
		Name:        "List",
		LocalConfig: userConfiguration,
		Cmd:         CmdListUsers,
		ArgsFn:      listArgsFn(sess.Info().TeamID, "users"),
	}
	return w.Run(ctx)
}

func listArgsFn(teamID string, prefix string) func() []string {
	return func() []string {
		if filename == "" {
			filename = makeFilename(prefix, teamID, extForType(commonFlags.listType))
		}
		return []string{filename}
	}
}

func wizChannels(ctx context.Context, _ *base.Command, _ []string) error {
	sess, err := bootstrap.SlackdumpSession(ctx)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return err
	}

	filename = makeFilename("channels", sess.Info().TeamID, extForType(commonFlags.listType))
	w := dumpui.Wizard{
		Title:       "List Channels",
		Name:        "List",
		LocalConfig: chanFlags.configuration,
		Cmd:         CmdListChannels,
		ArgsFn:      listArgsFn(sess.Info().TeamID, "channels"),
	}
	return w.Run(ctx)
}

func userConfiguration() cfgui.Configuration {
	c := cfgui.Configuration{
		cfgui.ParamGroup{
			Name: "User List Options",
			Params: []cfgui.Parameter{
				filenameParam("users" + extForType(commonFlags.listType)),
			},
		},
	}
	return append(c, commonFlags.configuration()...)
}

func filenameParam(placeholder string) cfgui.Parameter {
	return cfgui.Parameter{
		Name:        "Output Filename",
		Value:       filename,
		Description: "The filename to save the output to",
		Inline:      true,
		Updater:     updaters.NewFileNew(&filename, placeholder, false, true),
	}
}

func (o *channelOptions) configuration() cfgui.Configuration {
	c := cfgui.Configuration{
		cfgui.ParamGroup{
			Name: "Channel Options",
			Params: []cfgui.Parameter{
				filenameParam("channels" + extForType(commonFlags.listType)),
				{
					Name:        "Resolve Users",
					Value:       cfgui.Checkbox(o.resolveUsers),
					Description: "Resolve user IDs to names. Slow on large Slack workspaces.",
					Updater:     updaters.NewBool(&o.resolveUsers),
				},
			},
		},
		cfgui.ParamGroup{
			Name: "Cache Options",
			Params: []cfgui.Parameter{
				{
					Name:        "Disable Cache",
					Value:       cfgui.Checkbox(o.cache.Enabled),
					Description: "Disable channel cache",
					Updater:     updaters.NewBool(&o.cache.Enabled),
				},
				{
					Name:        "Cache Retention",
					Value:       o.cache.Retention.String(),
					Description: "Channel cache retention time. After this time, the cache is considered stale and will be refreshed.",
					Inline:      true,
					Updater:     updaters.NewDuration(&o.cache.Retention, false),
				},
				{
					Name:        "Cache Filename",
					Value:       o.cache.Filename,
					Description: "The filename of the cache",
					Inline:      true,
					Updater:     updaters.NewString(&o.cache.Filename, "channels.json", false, huh.ValidateNotEmpty()),
				},
			},
		},
	}
	return append(c, commonFlags.configuration()...)
}

func (l *commonOpts) configuration() cfgui.Configuration {
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
