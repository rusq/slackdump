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
	if err := huh.NewText().
		Title(msg).
		Description(help).
		Validate(validateFn).
		Value(&resp).
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
