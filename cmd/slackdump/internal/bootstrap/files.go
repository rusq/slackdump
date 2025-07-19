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
