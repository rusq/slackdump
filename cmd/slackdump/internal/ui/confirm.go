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

func FieldConfirm(b *bool, msg string, _ bool, opt ...Option) huh.Field {
	var opts = defaultOpts().apply(opt...)
	f := huh.NewConfirm().Title(msg).Description(opts.help).Value(b)
	return f
}
