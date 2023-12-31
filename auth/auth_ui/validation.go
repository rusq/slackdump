package auth_ui

import (
	"errors"
	"regexp"
)

var (
	ErrNotURLSafe = errors.New("not a valid url safe string")
	ErrRequired   = errors.New("can not be empty")
)

func valURLSafe(s string) error {
	for _, c := range s {
		if !isRuneURLSafe(c) {
			return ErrNotURLSafe
		}
	}
	return nil
}

func isRuneURLSafe(r rune) bool {
	switch {
	case 'a' <= r && r <= 'z':
		return true
	case 'A' <= r && r <= 'Z':
		return true
	case '0' <= r && r <= '9':
		return true
	case r == '-' || r == '.' || r == '_' || r == '~':
		return true
	}
	return false
}

func valRequired(s string) error {
	if s == "" {
		return ErrRequired
	}
	return nil
}

func valAND(fns ...func(string) error) func(string) error {
	return func(s string) error {
		for _, fn := range fns {
			if err := fn(s); err != nil {
				return err
			}
		}
		return nil
	}
}

var dumbEmailRE = regexp.MustCompile(`^[^@]+@[^@]+$`)

func valEmail(s string) error {
	if !dumbEmailRE.MatchString(s) {
		return errors.New("not a valid email")
	}
	return nil
}
