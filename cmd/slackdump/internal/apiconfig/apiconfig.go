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

const schemaJSONpath = "https://raw.githubusercontent.com/rusq/slackdump/cli-remake/cmd/slackdump/internal/apiconfig/schema.json"

var CmdConfig = &base.Command{
	UsageLine: "slackdump config",
	Short:     "API configuration",
	Long: `
# Config Command

Config command allows to perform different operations on the API limits
configuration file.
`,
	Commands: []*base.Command{
		CmdConfigNew,
		CmdConfigCheck,
	},
}

var ErrConfigInvalid = errors.New("config validation failed")

// Load reads, parses and validates the config file.
func Load(filename string) (slackdump.Limits, error) {
	f, err := os.Open(filename)
	if err != nil {
		return slackdump.Limits{}, err
	}
	defer f.Close()

	return readLimits(f)
}

// Save saves the config to the file.
func Save(filename string, limits slackdump.Limits) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return writeLimits(f, limits)
}

func readLimits(r io.Reader) (slackdump.Limits, error) {
	var limits slackdump.Limits
	dec := yaml.NewDecoder(r, yaml.DisallowUnknownField(), yaml.DisallowDuplicateKey())
	if err := dec.Decode(&limits); err != nil {
		return slackdump.Limits{}, err
	}

	if err := cfg.SlackConfig.Limits.Apply(limits); err != nil {
		if err := printErrors(os.Stderr, err); err != nil {
			return slackdump.Limits{}, err
		}
		return slackdump.Limits{}, ErrConfigInvalid
	}
	return limits, nil
}

func writeLimits(w io.Writer, cfg slackdump.Limits) error {
	fmt.Fprintf(w, "# yaml-language-server: $schema=%s\n", schemaJSONpath)
	return yaml.NewEncoder(w).Encode(cfg)
}

// printErrors prints configuration errors, if error is not nill and is of
// validator.ValidationErrors type.
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
