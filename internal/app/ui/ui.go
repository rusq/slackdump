// Package ui contains some common UI elements, that use Survey library.
package ui

import (
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
)

type inputOptions struct {
	stdio terminal.Stdio
	fileSelectorOpt
}

// surveyOpts returns the survey options.
func (io *inputOptions) surveyOpts() []survey.AskOpt {
	return []survey.AskOpt{
		survey.WithStdio(io.stdio.In, io.stdio.Out, io.stdio.Err),
	}
}

func (io *inputOptions) apply(opt ...Option) *inputOptions {
	for _, fn := range opt {
		fn(io)
	}
	return io
}

type Option func(*inputOptions)

func defaultOpts() *inputOptions {
	return &inputOptions{
		stdio: terminal.Stdio{
			In:  os.Stdin,
			Out: os.Stdout,
			Err: os.Stderr,
		},
	}
}

func WithOutput(stdio terminal.Stdio) Option {
	return func(io *inputOptions) {
		io.stdio = stdio
	}
}
