package apiconfig

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-yaml"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
)

var CmdConfig = &base.Command{
	UsageLine: "slackdump config",
	Short:     "API configuration",
	Long: base.Render(`
# Config Command

Config command allows to perform different operations on the API limits
configuration file.
`),
	Commands: []*base.Command{
		CmdConfigNew,
		CmdConfigCheck,
	},
}

var ErrConfigInvalid = errors.New("config validation failed")

// Load reads, parses and validates the config file.
func Load(filename string) (*slackdump.Options, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var opts slackdump.Options
	dec := yaml.NewDecoder(f, yaml.DisallowUnknownField(), yaml.DisallowDuplicateKey())
	if err := dec.Decode(&opts); err != nil {
		return nil, err
	}

	if err := cfg.SlackOptions.Apply(opts); err != nil {
		if err := printErrors(os.Stderr, err); err != nil {
			return nil, err
		}
		return nil, ErrConfigInvalid
	}
	return &opts, nil
}

func printErrors(w io.Writer, err error) error {
	if err == nil {
		return nil
	}

	var wErr error
	var printErr = func(format string, a ...any) {
		if wErr != nil {
			return
		}
		_, wErr = fmt.Fprintf(w, format, a...)
	}

	printErr("Detected problems:\n")
	var vErr validator.ValidationErrors
	if !errors.As(err, &vErr) {
		return err
	}
	for i, entry := range vErr {
		printErr("\t%2d: %s\n", i+1, entry.Translate(slackdump.OptErrTranslations))
	}
	return wErr
}
