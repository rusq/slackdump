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
package emoji

import (
	"context"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
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
					Value:       cfgui.Checkbox(o.full),
					Description: "Uses edge API to fetch full emoji information, including usernames",
					Updater:     updaters.NewBool(&o.full),
				},
			},
		},
		cfgui.ParamGroup{
			Name: "Download Options",
			Params: []cfgui.Parameter{
				{
					Name:        "Do Not Download",
					Value:       cfgui.Checkbox(cfg.WithFiles),
					Description: "Do not download, any emojis, just get the index",
					Updater:     updaters.NewBool(&cfg.WithFiles),
				},
				{
					Name:        "Ignore Download Errors",
					Value:       cfgui.Checkbox(o.FailFast),
					Description: "Ignore download errors and continue",
					Updater:     updaters.NewBool(&o.FailFast),
				},
			},
		},
	}
}
