package apiconfig

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/trace"

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
	_, task := trace.NewTask(ctx, "runConfigNew")
	defer task.End()

	if len(args) == 0 {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("config file name must be specified")
	}

	filename := maybeFixExt(args[0])

	if !shouldOverwrite(filename, *fNewOverride) {
		base.SetExitStatus(base.SUserError)
		return fmt.Errorf("file or directory exists: %q, use -y flag to overwrite (will not overwrite directory)", filename)
	}

	if err := Save(filename, slackdump.DefOptions.Limits); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("error writing the API limits config %q: %w", filename, err)
	}

	fmt.Printf("Your new API limits config is ready: %q\n", filename)
	return nil
}

// shouldOverwrite returns true if the file can be overwritten.  If override
// is true and the file exists and not a directory, it will return true.
func shouldOverwrite(filename string, override bool) bool {
	fi, err := os.Stat(filename)
	if fi != nil && fi.IsDir() {
		return false
	}
	return err != nil || override
}

// maybeFixExt checks if the extension is one of .yaml or .yml, and if not
// appends it to teh file.
func maybeFixExt(filename string) string {
	if ext := filepath.Ext(filename); !(ext == ".yaml" || ext == ".yml") {
		return maybeAppendExt(filename, ".yaml")
	}
	return filename
}

// maybeAppendExt adds a filename extension ext if the filename has missing, or
// a different extension.
func maybeAppendExt(filename string, ext string) string {
	if len(ext) == 0 {
		return filename
	}
	if ext[0] != '.' {
		ext = "." + ext
	}
	if filepath.Ext(filename) == ext {
		return filename
	}
	return filename + ext
}
