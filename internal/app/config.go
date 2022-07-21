package app

import (
	"errors"
	"fmt"
	"html/template"
	"strings"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/types"
)

const (
	filenameTmplName = "fnt"
)

type Config struct {
	ListFlags ListFlags

	Input  Input  // parameters of the input
	Output Output // " " output

	Oldest TimeValue // oldest time to dump conversations from
	Latest TimeValue // latest time to dump conversations to

	FilenameTemplate string

	ExportName string

	Options slackdump.Options
}

type Output struct {
	Filename string
	Format   string // output format
	Base     string // base directory or zip file
}

type Input struct {
	List *structures.EntityList // Include channels
}

var (
	ErrInvalidInput = errors.New("no valid input")

	errSkip = errors.New("skip")
)

func (in *Input) IsValid() bool {
	return !in.List.IsEmpty()
}

// listProducer iterates over the input.List.Include, and calls fn for each
// entry.
func (in *Input) listProducer(fn func(string) error) error {
	if !in.List.HasIncludes() {
		return ErrInvalidInput
	}
	for _, entry := range in.List.Include {
		if err := fn(entry); err != nil {
			if errors.Is(err, errSkip) {
				continue
			}
			return err
		}
	}
	return nil
}

func (out Output) FormatValid() bool {
	return out.Format != "" && (out.Format == OutputTypeJSON ||
		out.Format == OutputTypeText)
}

func (out Output) IsText() bool {
	return out.Format == OutputTypeText
}

type ListFlags struct {
	Users    bool
	Channels bool
}

func (lf ListFlags) FlagsPresent() bool {
	return lf.Users || lf.Channels
}

var ErrNothingToDo = errors.New("no valid input and no list flags specified")

// Validate checks if the command line parameters have valid values.
func (p *Config) Validate() error {
	if p.ExportName != "" {
		// slack workspace export mode.
		return nil
	}

	if !p.Input.IsValid() && !p.ListFlags.FlagsPresent() {
		return ErrNothingToDo
	}

	// channels and users listings will be in the text format (if not specified otherwise)
	if p.Output.Format == "" {
		if p.ListFlags.FlagsPresent() {
			p.Output.Format = OutputTypeText
		} else {
			p.Output.Format = OutputTypeJSON
		}
	}

	if !p.ListFlags.FlagsPresent() && !p.Output.FormatValid() {
		return fmt.Errorf("invalid Output type: %q, must use one of %v", p.Output.Format, []string{OutputTypeJSON, OutputTypeText})
	}

	// validate file naming template
	if err := p.compileValidateTemplate(); err != nil {
		return err
	}

	return nil
}

func (cfg *Config) compileTemplates() (*template.Template, error) {
	return template.New(filenameTmplName).Parse(cfg.FilenameTemplate)
}

func (cfg *Config) compileValidateTemplate() error {
	tmpl, err := cfg.compileTemplates()
	if err != nil {
		return err
	}
	// are you ready for some filth? Here we go!

	// let's define some indicators
	const (
		NotOK     = "$$ERROR$$"   // not allowed at all
		OK        = "$$OK$$"      // required
		PartialOK = "$$PARTIAL$$" // partial (only goes well with OK)
	)

	// marking all the fields we want with OK, all the rest (the ones we DO NOT
	// WANT) with NotOK.
	tc := types.Conversation{
		Name:     OK,
		ID:       OK,
		Messages: []types.Message{{Message: slack.Message{Msg: slack.Msg{Channel: NotOK}}}},
		ThreadTS: PartialOK,
	}

	// now we render the template and check for OK/NotOK values in the output.
	var buf strings.Builder
	if err := tmpl.ExecuteTemplate(&buf, filenameTmplName, tc); err != nil {
		return err
	}
	if strings.Contains(buf.String(), NotOK) || len(buf.String()) == 0 {
		return fmt.Errorf("invalid fields in the template: %q", cfg.FilenameTemplate)
	}
	if !strings.Contains(buf.String(), OK) {
		// must contain at least one OK
		return fmt.Errorf("this does not resolve to anything useful: %q", cfg.FilenameTemplate)
	}
	return nil
}

// Producer iterates over the list or reads the list from the file and calls
// fn for each entry.
func (in Input) producer(fn func(string) error) error {
	if !in.IsValid() {
		return ErrInvalidInput
	}

	return in.listProducer(fn)
}
