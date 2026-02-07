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
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/go-playground/validator/v10"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/network"
)

var CmdConfig = &base.Command{
	UsageLine: "slackdump config",
	Short:     "API configuration",
	Long: `
# Config Command

Config command allows to perform different operations on the API limits
configuration file.
`,
	Commands: []*base.Command{
		CmdConfigCheck,
		CmdConfigNew,
	},
}

var ConfigExts = []string{"*.toml", "*.tml"}

var ErrConfigInvalid = errors.New("config validation failed")

// Load reads, parses and validates the config file.
func Load(filename string) (network.Limits, error) {
	f, err := os.Open(filename)
	if err != nil {
		return network.Limits{}, err
	}
	defer f.Close()

	return applyLimits(f)
}

// Save saves the config to the file.
func Save(filename string, limits network.Limits) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return writeLimits(f, limits)
}

// applyLimits reads the limits from the reader, validates them and applies to
// the global config.
func applyLimits(r io.Reader) (network.Limits, error) {
	var limits network.Limits
	dec := toml.NewDecoder(r)
	if _, err := dec.Decode(&limits); err != nil {
		return network.Limits{}, err
	}

	if err := cfg.Limits.Apply(limits); err != nil {
		if err := printErrors(os.Stderr, err); err != nil {
			return network.Limits{}, err
		}
		return network.Limits{}, ErrConfigInvalid
	}
	return limits, nil
}

func writeLimits(w io.Writer, cfg network.Limits) error {
	return toml.NewEncoder(w).Encode(cfg)
}

// printErrors prints configuration errors, if error is not nill and is of
// validator.ValidationErrors type.
func printErrors(w io.Writer, err error) error {
	if err == nil {
		return nil
	}

	var wErr error
	printErr := func(format string, a ...any) {
		if wErr != nil {
			return
		}
		_, wErr = fmt.Fprintf(w, format, a...)
	}

	printErr("Detected problems:\n")
	var vErr validator.ValidationErrors
	if !errors.As(err, &vErr) {
		return err
	}
	for i, entry := range vErr {
		printErr("\t%2d: %s\n", i+1, entry.Translate(network.OptErrTranslations))
	}
	return wErr
}
