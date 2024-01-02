package ask

import (
	"github.com/charmbracelet/huh"
	"github.com/rusq/slackdump/v2/export"
)

// ExportType asks the user to select an export type.
func ExportType() (export.ExportType, error) {
	var resp export.ExportType
	q := huh.NewSelect[export.ExportType]().
		Title("Export type: ").
		Options(
			huh.NewOption("Mattermost   - bulk upload compatible export (see doc)", export.TMattermost),
			huh.NewOption("Standard     - standard export format", export.TStandard),
		).
		Value(&resp)
	if err := q.Run(); err != nil {
		return 0, err
	}
	return resp, nil
}
