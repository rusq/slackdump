package ui

import (
	"errors"

	"github.com/charmbracelet/huh"
)

// Input shows a text input field with a custom validator.
func Input(msg, help string, validator func(s string) error) (string, error) {
	if validator == nil {
		validator = noValidation
	}
	var resp string
	if err := huh.NewText().
		Title(msg).
		Description(help).
		Validate(validator).
		Value(&resp).
		Run(); err != nil {
		return "", err
	}
	return resp, nil
}

// StringRequire requires user to input string.
func StringRequire(msg, help string) (string, error) {
	return Input(msg, help, func(s string) error {
		if s == "" {
			return errors.New("value is required")
		}
		return nil
	})
}

// String asks user to input string, accepts an empty input.
func String(msg, help string) (string, error) {
	return Input(msg, help, noValidation)
}

func noValidation(s string) error {
	return nil
}
