// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package list

import (
	"context"

	"github.com/charmbracelet/huh"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui/cfgui"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui/dumpui"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui/updaters"
	"github.com/rusq/slackdump/v4/internal/format"
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
				cfgui.ChannelTypes(),
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
				{
					Name:        "Bare Format",
					Value:       cfgui.Checkbox(l.bare),
					Description: "Use bare format: just the user or channel ID, no headers",
					Updater:     updaters.NewBool(&l.bare),
				},
			},
		},
	}
	return c
}
