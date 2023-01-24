package ask

import (
	"fmt"
	"strings"

	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/internal/ui"
)

// ConversationList asks the user for the list of conversations to dump or
// export. msg is the message to display to the user.
func ConversationList(msg string) (*structures.EntityList, error) {
	for {
		chanStr, err := ui.String(
			msg,
			"Enter whitespace separated conversation IDs or URLs to export.\n"+
				"   - prefix with ^ (caret) to exclude the converation\n"+
				"   - prefix with @ to read the list of converations from the file.\n\n"+
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
