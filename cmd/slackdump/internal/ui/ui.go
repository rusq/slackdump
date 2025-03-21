// Package ui contains some common UI elements.
package ui

const (
	// MenuSeparator is the separator to use in the wizard menus.
	MenuSeparator = "────────────────"
)

type inputOptions struct {
	fileSelectorOpt
	help string
}

func (io *inputOptions) apply(opt ...Option) *inputOptions {
	for _, fn := range opt {
		fn(io)
	}
	return io
}

type Option func(*inputOptions)

func defaultOpts() *inputOptions {
	return &inputOptions{}
}

// WithHelp sets the help message.
func WithHelp(msg string) Option {
	return func(io *inputOptions) {
		io.help = msg
	}
}
