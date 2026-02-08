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
package dump

import (
	"context"

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
			return structures.SplitEntryList(entryList)
		},
		ValidateParamsFn: func() error {
			return structures.ValidateEntityList(entryList)
		},
	}
	return w.Run(ctx)
}

var entryList string

func (o *options) configuration() cfgui.Configuration {
	return cfgui.Configuration{
		{
			Name: "Required",
			Params: []cfgui.Parameter{
				cfgui.ChannelIDs(&entryList, true),
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
					Name:        "Update links",
					Value:       cfgui.Checkbox(o.updateLinks),
					Description: "Update file links to point to the downloaded files",
					Updater:     updaters.NewBool(&o.updateLinks),
				},
			},
		},
	}
}
