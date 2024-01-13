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

// valSepEaster is probably the most useless validation function ever.
func valSepEaster() func(v LoginType) error {
	var phrases = []string{
		"This is a separator, it does nothing",
		"Seriously, it does nothing",
		"Stop clicking on it",
		"Stop it",
		"Stop",
		"Why are you so persistent?",
		"Fine, you win",
		"Here's a cookie: 🍪",
		"🍪",
		"🍪",
		"Don't be greedy, you already had three.",
		"Ok, here's another one: 🍪",
		"Nothing will happen if you click on it again",
		"",
		"",
		"",
		"You must have a lot of time on your hands",
		"Or maybe you're just bored",
		"Or maybe you're just procrastinating",
		"Or maybe you're just trying to get a cookie",
		"These are virtual cookies, you can't eat them, but here's another one: 🍪",
		"🍪",
		"You have reached the end of this joke, it will now repeat",
		"Seriously...",
		"Ah, shit, here we go again",
	}
	var i int
	return func(v LoginType) error {
		if v == -1 {
			// separator selected
			msg := phrases[i]
			i = (i + 1) % len(phrases)
			return errors.New(msg)
		}
		return nil
	}
}
