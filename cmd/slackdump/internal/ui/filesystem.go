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
package ui

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
)

type fileSelectorOpt struct {
	defaultFilename string // if set, the empty filename will be replaced to this value
	mustExist       bool
}

func WithDefaultFilename(s string) Option {
	return func(io *inputOptions) {
		io.fileSelectorOpt.defaultFilename = s
	}
}

func WithMustExist(b bool) Option {
	return func(io *inputOptions) {
		io.mustExist = b
	}
}

func FileSelector(msg, descr string, opt ...Option) (string, error) {
	var opts = defaultOpts().apply(opt...)

	var resp struct {
		Filename string
	}
	q := huh.NewForm(huh.NewGroup(fieldFileInput(&resp.Filename, msg, descr, *opts))).WithTheme(HuhTheme())

	for {
		if err := q.Run(); err != nil {
			return "", err
		}
		if resp.Filename == "" && opts.defaultFilename != "" {
			resp.Filename = opts.defaultFilename
		}
		if opts.mustExist {
			break
		}
		if _, err := os.Stat(resp.Filename); err != nil {
			break
		}
		overwrite, err := Confirm(fmt.Sprintf("File %q exists. Overwrite?", resp.Filename), false, opt...)
		if err != nil {
			return "", err
		}
		if overwrite {
			break
		}
	}
	return resp.Filename, nil
}

func checkExists(filename string) error {
	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			return errors.New("file must exist")
		} else {
			return err
		}
	}
	return nil
}

func FieldFileInput(filename *string, msg, descr string, opt ...Option) huh.Field {
	var opts = defaultOpts().apply(opt...)
	return fieldFileInput(filename, msg, descr, *opts)
}

func fieldFileInput(filename *string, msg, descr string, opts inputOptions) huh.Field {
	q := huh.NewInput().
		Title(msg).
		Description(descr).
		Value(filename).
		Validate(func(ans string) error {
			filename := ans
			if filename == "" {
				if opts.defaultFilename == "" {
					return errors.New("empty filename")
				} else {
					if !opts.mustExist {
						return nil
					} else {
						return checkExists(opts.defaultFilename)
					}
				}
			}
			if opts.mustExist {
				return checkExists(filename)
			}
			return nil
		})
	return q
}
