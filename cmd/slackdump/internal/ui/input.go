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

// Input shows a text input field with a custom validator.
func Input(msg, help string, validateFn func(s string) error) (string, error) {
	if validateFn == nil {
		validateFn = NoValidation
	}
	var resp string
	if err := huh.NewForm(huh.NewGroup(huh.NewText().
		Title(msg).
		Description(help).
		Validate(validateFn).
		Value(&resp))).WithTheme(HuhTheme()).
		Run(); err != nil {
		return "", err
	}
	return resp, nil
}

// StringRequire requires user to input string.
func StringRequire(msg, help string) (string, error) {
	return Input(msg, help, ValidateNotEmpty)
}

// String asks user to input string, accepts an empty input.
func String(msg, help string) (string, error) {
	return Input(msg, help, NoValidation)
}
