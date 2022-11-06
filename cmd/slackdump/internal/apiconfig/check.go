package apiconfig

import (
	"context"
	"errors"
	"fmt"

	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
)

var CmdConfigCheck = &base.Command{
	UsageLine: "slackdump config check",
	Short:     "validate the existing config for errors",
	Long: base.Render(`
# Config Check Command

Allows to check the config for errors and invalid values.

Example:

    slackdump config check myconfig.yaml

It will check for duplicate and unknown keys, and also ensure that values are
within the allowed boundaries.
`),
}

func init() {
	CmdConfigCheck.Run = runConfigCheck
}

func runConfigCheck(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) == 0 || args[0] == "" {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("config filename must be specified")
	}
	filename := args[0]
	if _, err := Load(filename); err != nil {
		base.SetExitStatus(base.SUserError)
		return err
	}
	fmt.Printf("Config file %q: OK\n", filename)
	return nil
}
