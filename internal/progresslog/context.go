// Package progresslog carries optional, progress-only console descriptions.
package progresslog

import (
	"context"
	"strings"
	"unicode"

	"github.com/charmbracelet/x/ansi"
)

type messageKey struct{}

// WithMessage sets a console-only message; the original log record is unchanged.
func WithMessage(ctx context.Context, message string) context.Context {
	return context.WithValue(ctx, messageKey{}, message)
}

// Message returns the progress-only console message, if present.
func Message(ctx context.Context) (string, bool) {
	message, ok := ctx.Value(messageKey{}).(string)
	return message, ok
}

// Sanitize removes terminal escapes and makes a message safe for one line.
func Sanitize(message string) string {
	return strings.Map(func(c rune) rune {
		if unicode.IsControl(c) || unicode.Is(unicode.Zl, c) || unicode.Is(unicode.Zp, c) {
			return ' '
		}
		return c
	}, ansi.Strip(message))
}
