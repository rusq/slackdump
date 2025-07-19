package bootstrap

import (
	"fmt"
	"os"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
)

var yesno = base.YesNo

// AskOverwrite checks if the given path and:
//   - if YesMan flag is set to true - returns nil;
//   - if path does not exist, it returns nil;
//   - if path exists, it prompts the user to confirm
//     overwriting it, and if the user confirms, it returns nil, otherwise it
//     returns an error.
func AskOverwrite(path string) error {
	if cfg.YesMan {
		return nil
	}
	if _, err := os.Stat(path); err == nil {
		if !yesno(fmt.Sprintf("Output path %q already exists. Overwrite?", path)) {
			base.SetExitStatus(base.SCancelled)
			return base.ErrOpCancelled
		}
	}
	return nil
}
