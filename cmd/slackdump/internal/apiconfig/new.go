package apiconfig

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
)

var CmdConfigNew = &base.Command{
	UsageLine: "slackdump config new",
	Short:     "creates a new API config with the default values",
	Long: base.Render(`
# Config New Command

Creates a new API configuration file containing default values. You will need
to specify the filename, for example:

    slackdump config new myconfig.yaml

If the extension is omitted, ".yaml" is automatically appended to the filename.
`),
	FlagMask:   cfg.OmitAll,
	PrintFlags: true,
}

var (
	fNewOverride = CmdConfigNew.Flag.Bool("y", false, "confirm the overwrite of the existing config")
)

func init() {
	CmdConfigNew.Run = runConfigNew
}

func runConfigNew(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) == 0 {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("config file name must be specified")
	}

	filename := args[0]
	if ext := filepath.Ext(filename); !(ext == ".yaml" || ext == ".yml") {
		filename = maybeAddExt(filename, ".yaml")
	}

	if _, err := os.Stat(filename); !*fNewOverride && err == nil {
		base.SetExitStatus(base.SUserError)
		return fmt.Errorf("refusing to overwrite file %q, use -y flag to overwrite", filename)
	}

	if err := Save(filename, &slackdump.DefOptions.Limits); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("error writing the API config %q: %w", filename, err)
	}

	fmt.Printf("Your new API config is ready: %q\n", filename)
	return nil
}

// maybeAddExt adds a filename extension ext if the filename has missing, or
// a different extension.
func maybeAddExt(filename string, ext string) string {
	if len(ext) == 0 {
		return filename
	}
	if filepath.Ext(filename) == ext {
		return filename
	}
	if ext[0] != '.' {
		ext = "." + ext
	}
	return filename + ext
}
