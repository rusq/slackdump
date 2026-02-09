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

package apiconfig

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"path/filepath"
	"runtime/trace"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui"
	"github.com/rusq/slackdump/v4/internal/network"
)

//go:embed assets/config_new.md
var configNewMD string

var CmdConfigNew = &base.Command{
	UsageLine:  "slackdump config new",
	Short:      "creates a new API config with the default values",
	Long:       configNewMD,
	FlagMask:   cfg.OmitAll &^ cfg.OmitYesManFlag,
	PrintFlags: true,
}

func init() {
	CmdConfigNew.Run = runConfigNew
	CmdConfigNew.Wizard = wizConfigNew
}

func runConfigNew(ctx context.Context, cmd *base.Command, args []string) error {
	_, task := trace.NewTask(ctx, "runConfigNew")
	defer task.End()

	if len(args) == 0 {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("config file name must be specified")
	}

	filename := maybeFixExt(args[0])

	if err := bootstrap.AskOverwrite(filename); err != nil {
		return err
	}

	if err := Save(filename, network.DefLimits); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("error writing the API limits config %q: %w", filename, err)
	}

	_, err := printConfigOK(filename)
	return err
}

// printConfigOK outputs the confirmation message to the user.
func printConfigOK(filename string) (n int, err error) {
	return fmt.Printf("Your new API limits config is ready: %q\n", filename)
}

func wizConfigNew(ctx context.Context, cmd *base.Command, args []string) error {
	ctx, task := trace.NewTask(ctx, "wizConfigNew")
	defer task.End()

RESTART:
	filename, err := ui.FileSelector("New config file name", "Enter new limiter config file name")
	if err != nil {
		return err
	}
	filename = maybeFixExt(filename)
	if err := Save(filename, network.DefLimits); err != nil {
		fmt.Printf("Error: %s, please retry\n", err)
		trace.Logf(ctx, "error", "error saving file to %q: %s, survey restarted", filename, err)
		goto RESTART
	}

	_, err = printConfigOK(filename)
	return err
}

// maybeFixExt checks if the extension is one of .toml or .tml, and if not
// appends it to the file.
func maybeFixExt(filename string) string {
	if ext := filepath.Ext(filename); !(ext == ".toml" || ext == ".tml") {
		return maybeAppendExt(filename, ".toml")
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
