package app

import (
	"bufio"
	"errors"
	"fmt"
	"html/template"
	"io"
	"os"
	"strings"

	"github.com/rusq/slackdump"
	"github.com/slack-go/slack"
)

const filenameTmplName = "fnt"

type Config struct {
	Creds     SlackCreds
	ListFlags ListFlags

	Input  Input  // parameters of the input
	Output Output // " " output

	Oldest TimeValue // oldest time to dump conversations from
	Latest TimeValue // latest time to dump conversations to

	FilenameTemplate string

	Options slackdump.Options
}

type Output struct {
	Filename string
	Format   string
}

type Input struct {
	List     []string // Input list
	Filename string   // filename containing the list of Conversation IDs or URLs to download.
}

var (
	ErrInvalidInput = errors.New("no valid input")

	errSkip = errors.New("skip")
)

func (in Input) IsValid() bool {
	return len(in.List) > 0 || in.Filename != ""
}

// listProducer iterates over the input.List, and calls fn for each entry.
func (in Input) listProducer(fn func(string) error) error {
	if !in.IsValid() {
		return ErrInvalidInput
	}
	for _, entry := range in.List {
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

// SlackCreds stores Slack credentials.
type SlackCreds struct {
	// Token is the Slack auth-token value.
	Token string
	// Cookie may contain cookie value or a filename.  I have mixed feelings
	// about this, but blame cURL, they use the same approach for --cookie, and
	// I have no imagination.  Also, it's simpler from the usability POV, as
	// there's no additional parameter.  Well, I'll repeat this to myself until
	// I start to believe in it.
	Cookie string
}

func (c SlackCreds) Validate() error {
	if c.Token == "" {
		return fmt.Errorf("token is not specified")
	} else if c.Cookie == "" {
		return fmt.Errorf("cookie is not specified")
	}

	stat, err := os.Stat(c.Cookie)
	if err != nil {
		if os.IsNotExist(err) {
			// cookie is not a file
			return nil
		}
		return fmt.Errorf("cookie file: stat error: %w", err)
	}

	if stat.IsDir() {
		return fmt.Errorf("cookie path is a directory")
	}

	// cookie is a file, try to open it for reading
	f, err := os.Open(c.Cookie)
	if err != nil {
		return fmt.Errorf("error opening the cookie file: %w", err)
	}
	f.Close()

	return nil
}

type ListFlags struct {
	Users    bool
	Channels bool
}

func (lf ListFlags) FlagsPresent() bool {
	return lf.Users || lf.Channels
}

// Validate checks if the command line parameters have valid values.
func (p *Config) Validate() error {
	if err := p.Creds.Validate(); err != nil {
		return err
	}

	if !p.Input.IsValid() && !p.ListFlags.FlagsPresent() {
		return fmt.Errorf("no valid input and no list flags specified")
	}
	p.Creds.Cookie = strings.TrimPrefix(p.Creds.Cookie, "d=")

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
	tc := slackdump.Conversation{
		Name:     OK,
		ID:       OK,
		Messages: []slackdump.Message{{Message: slack.Message{Msg: slack.Msg{Channel: NotOK}}}},
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
	if in.Filename != "" {
		return in.fileProducer(fn)
	} else {
		return in.listProducer(fn)
	}
}

// fileProducer iterates over the file, reading it line by line, and calls fn
// for each line.
func (in Input) fileProducer(fn func(string) error) error {
	if !in.IsValid() {
		return ErrInvalidInput
	}
	f, err := openFile(in.Filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return in.iterScanner(f, fn)
}

// iterScanner iterates over the reader r, reading it line by line, and calls fn
// for each line.
func (in Input) iterScanner(r io.Reader, fn func(string) error) error {
	if !in.IsValid() {
		return ErrInvalidInput
	}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if err := fn(scanner.Text()); err != nil {
			if errors.Is(err, errSkip) {
				continue
			}
			return err
		}
	}
	return scanner.Err()
}
