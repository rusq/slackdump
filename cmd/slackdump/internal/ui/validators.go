package ui

import "errors"

func NoValidation(s string) error {
	return nil
}

func ValidateNotEmpty(s string) error {
	if s == "" {
		return errors.New("value is required")
	}
	return nil
}
