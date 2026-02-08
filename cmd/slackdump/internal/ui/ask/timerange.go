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
package ask

import (
	"errors"
	"time"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui"
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
	if latest, err = ui.Time("Latest message"); err != nil && !errors.Is(err, ui.ErrEmptyOptionalInput) {
		return
	}
	err = nil
	return
}
