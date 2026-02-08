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
package export

import (
	"context"

	"github.com/charmbracelet/huh"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui/cfgui"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui/dumpui"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui/updaters"
	"github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/source"
)

func wizExport(ctx context.Context, cmd *base.Command, args []string) error {
	w := &dumpui.Wizard{
		Title:       "Export Slack Workspace",
		Name:        "Export",
		Cmd:         cmd,
		LocalConfig: options.configuration,
		ArgsFn: func() []string {
			if len(entryList) > 0 {
				return structures.SplitEntryList(entryList)
			}
			return nil
		},
	}
	return w.Run(ctx)
}

var entryList string

func (fl *exportFlags) configuration() cfgui.Configuration {
	return cfgui.Configuration{
		{
			Name: "Optional",
			Params: []cfgui.Parameter{
				cfgui.ChannelIDs(&entryList, false),
				{
					Name:        "Export Storage Type",
					Value:       fl.ExportStorageType.String(),
					Description: "Export file storage type",
					Inline:      false,
					Updater: updaters.NewPicklist(&fl.ExportStorageType, huh.NewSelect[source.StorageType]().
						Title("Choose File storage type").
						Options(
							huh.NewOption("Mattermost", source.STmattermost),
							huh.NewOption("Standard", source.STstandard),
							huh.NewOption("Disable", source.STnone),
						)),
				},
				cfgui.MemberOnly(),
				cfgui.OnlyChannelUsers(),
				cfgui.IncludeCustomLabels(),
				cfgui.Avatars(),
				{
					Name:        "Export Token",
					Value:       fl.ExportToken,
					Description: "File export token to append to each of the file URLs",
					Inline:      true,
					Updater:     updaters.NewString(&fl.ExportToken, "", false, structures.ValidateToken),
				},
				cfgui.ChannelTypes(),
			},
		},
	}
}
