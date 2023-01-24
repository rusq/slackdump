package ask

import (
	"errors"
	"time"

	"github.com/rusq/slackdump/v2/internal/ui"
)

func MaybeTimeRange() (oldest, latest time.Time, err error) {
	// ask if user wants time range
	needRange, err := ui.Confirm(
		"Do you want to specify the time range?",
		false,
		ui.WithHelp("If you don't specify the time range, the entire history will be exported.\nIf you need to skip one of the time range values, leave date empty and press Enter."))
	if err != nil || !needRange {
		return
	}
	return TimeRange()
}

// TimeRange asks for a time range.
func TimeRange() (oldest, latest time.Time, err error) {
	// ask for the time range
	if oldest, err = ui.Time("Earliest message"); err != nil && !errors.Is(err, ui.ErrEmptyOptionalInput) {
		return
	}
	err = nil
	if latest, err = ui.Time("Latest message"); err != nil && !errors.Is(err, ui.ErrEmptyOptionalInput) {
		return
	}
	err = nil
	return
}
