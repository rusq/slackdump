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
	"github.com/charmbracelet/huh"
)

func Confirm(msg string, _ bool, opt ...Option) (bool, error) {
	var b bool
	if err := FieldConfirm(&b, msg, false, opt...).Run(); err != nil {
		return false, err
	}
	return b, nil
}

func FieldConfirm(b *bool, msg string, _ bool, opt ...Option) *huh.Form {
	var opts = defaultOpts().apply(opt...)
	f := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title(msg).Description(opts.help).Value(b),
	)).WithTheme(HuhTheme()).WithKeyMap(DefaultHuhKeymap)
	return f
}
