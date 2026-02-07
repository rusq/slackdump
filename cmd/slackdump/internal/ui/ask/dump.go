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
	"fmt"
	"strings"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
	"github.com/rusq/slackdump/v3/internal/structures"
)

// ConversationList asks the user for the list of conversations to dump or
// export. msg is the message to display to the user.
func ConversationList(msg string) (*structures.EntityList, error) {
	for {
		chanStr, err := ui.String(
			msg,
			"Enter whitespace separated conversation IDs or URLs to export.\n"+
				"   - prefix with ^ (caret) to exclude the conversation\n"+
				"   - prefix with @ to read the list of conversations from the file.\n\n"+
				"For more details, see https://github.com/rusq/slackdump/blob/master/doc/usage-export.rst#providing-the-list-in-a-file",
		)
		if err != nil {
			return nil, err
		}
		if chanStr == "" || strings.ToLower(chanStr) == "all" {
			return new(structures.EntityList), nil
		}
		if el, err := structures.NewEntityList(strings.Split(chanStr, " ")); err != nil {
			fmt.Println(err)
		} else {
			return el, nil
		}
	}
}
