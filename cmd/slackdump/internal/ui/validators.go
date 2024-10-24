package ui

import (
	"errors"
	"os"
)

func NoValidation(s string) error {
	return nil
}

func ValidateNotEmpty(s string) error {
	if s == "" {
		return errors.New("value is required")
	}
	return nil
}

func ValidateNotExists(s string) error {
	_, err := os.Stat(s)
	if err == nil {
		return errors.New("file already exists")
	}
	return nil
}
