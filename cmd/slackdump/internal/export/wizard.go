package export

import (
	"github.com/AlecAivazis/survey/v2"
	"github.com/rusq/slackdump/v2/export"
)

// AskExportType asks the user to select an export type.
func AskExportType() (export.ExportType, error) {
	mode := &survey.Select{
		Message: "Export type: ",
		Options: []string{export.TMattermost.String(), export.TStandard.String()},
		Description: func(value string, index int) string {
			descr := []string{
				"Mattermost bulk upload compatible export (see doc)",
				"Standard export format",
			}
			return descr[index]
		},
	}
	var resp string
	if err := survey.AskOne(mode, &resp); err != nil {
		return 0, err
	}
	var t export.ExportType
	t.Set(resp)
	return t, nil
}
