package export

import (
	"context"

	"github.com/charmbracelet/huh"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/cfgui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/dumpui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/updaters"
	"github.com/rusq/slackdump/v3/internal/chunk/transform/fileproc"
	"github.com/rusq/slackdump/v3/internal/structures"
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
					Updater: updaters.NewPicklist(&fl.ExportStorageType, huh.NewSelect[fileproc.StorageType]().
						Title("Choose File storage type").
						Options(
							huh.NewOption("Mattermost", fileproc.STmattermost),
							huh.NewOption("Standard", fileproc.STstandard),
							huh.NewOption("Disable", fileproc.STnone),
						)),
				},
				cfgui.MemberOnly(),
				cfgui.Avatars(),
				{
					Name:        "Export Token",
					Value:       fl.ExportToken,
					Description: "File export token to append to each of the file URLs",
					Inline:      true,
					Updater:     updaters.NewString(&fl.ExportToken, "", false, structures.ValidateToken),
				},
			},
		},
	}
}
