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
// Package ui contains some common UI elements.
package ui

const (
	// MenuSeparator is the separator to use in the wizard menus.
	MenuSeparator = "────────────────"
)

type inputOptions struct {
	fileSelectorOpt
	help string
}

func (io *inputOptions) apply(opt ...Option) *inputOptions {
	for _, fn := range opt {
		fn(io)
	}
	return io
}

type Option func(*inputOptions)

func defaultOpts() *inputOptions {
	return &inputOptions{}
}

// WithHelp sets the help message.
func WithHelp(msg string) Option {
	return func(io *inputOptions) {
		io.help = msg
	}
}
