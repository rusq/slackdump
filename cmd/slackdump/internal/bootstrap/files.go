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
package bootstrap

import (
	"fmt"
	"os"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/osext"
)

var yesno = base.YesNo

func init() {
	if !osext.IsInteractive() {
		yesno = func(_ string) bool {
			return true // assume yes in non-interactive mode, otherwise gets stuck
		}
	}
}

// AskOverwrite checks if the given path and:
//   - if [cfg.YesMan] flag is set to true - returns nil;
//   - if path does not exist, it returns nil;
//   - if path exists, it prompts the user to confirm
//     overwriting it, and if the user confirms, it returns nil, otherwise it
//     returns [base.ErrOpCancelled]
func AskOverwrite(path string) error {
	if cfg.YesMan {
		return nil
	}
	if _, err := os.Stat(path); err == nil {
		if !yesno(fmt.Sprintf("Path %q already exists. Overwrite", path)) {
			base.SetExitStatus(base.SCancelled)
			return base.ErrOpCancelled
		}
	}
	return nil
}
