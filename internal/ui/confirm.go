package ui

import (
	"github.com/charmbracelet/huh"
)

func Confirm(msg string, _ bool, opt ...Option) (bool, error) {
	var opts = defaultOpts().apply(opt...)

	var b bool
	if err := huh.NewConfirm().Title(msg).Description(opts.help).Value(&b).Run(); err != nil {
		return false, err
	}
	return b, nil
}
