package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rusq/dlog"
	"github.com/rusq/slackdump"
)

const (
	OutputTypeJSON = "json"
	OutputTypeText = "text"
)

type App struct {
	sd *slackdump.SlackDumper

	cfg Config
}

// New creates a new slackdump app.
func New(cfg Config) (*App, error) {
	return &App{cfg: cfg}, nil
}

// init initialises the slack dumper app.
func (app *App) init(ctx context.Context) error {
	sd, err := slackdump.New(
		ctx,
		app.cfg.Creds.Token,
		app.cfg.Creds.Cookie,
		slackdump.DumpFiles(app.cfg.IncludeFiles),
		slackdump.LimiterBoost(app.cfg.Boost),
	)
	if err != nil {
		return err
	}
	app.sd = sd
	return nil
}

func (app *App) Run(ctx context.Context) error {
	if err := app.init(ctx); err != nil {
		return err
	}

	if app.cfg.ListFlags.FlagsPresent() {
		return app.listEntities(ctx)
	}

	n, err := app.dumpChannels(ctx, app.cfg.ChannelIDs)
	if err != nil {
		return err
	}
	dlog.Printf("job finished, dumped %d channels", n)
	return nil
}

// Validate checks if the command line parameters have valid values.
func (p *Config) Validate() error {
	if !p.Creds.Valid() {
		return fmt.Errorf("slack token or cookie not specified")
	}

	if len(p.ChannelIDs) == 0 && !p.ListFlags.FlagsPresent() {
		return fmt.Errorf("no ListFlags flags specified and no channels to export")
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

	return nil
}

// dumpChannels dumps the channels with ids, if dumpfiles is true, it will save
// the files into a respective directory with ID of the channel as the name.  If
// generateText is true, it will additionally format the conversation as text
// file and write it to <ID>.txt file.
//
// The result of the work of this function, for each channel ID, the following
// files will be created:
//
//    +-<ID> - directory, if dumpfiles is true
//    |  +- attachment1.ext
//    |  +- attachment2.ext
//    |  +- ...
//    +--<ID>.json - json file with conversation and users
//    +--<ID>.txt  - formatted conversation in text format, if generateText is true.
//
func (app *App) dumpChannels(ctx context.Context, ids []string) (int, error) {
	var total int
	for _, ch := range ids {
		dlog.Printf("dumping channel: %q", ch)

		if err := app.dumpOneChannel(ctx, ch); err != nil {
			dlog.Printf("channel %q: %s", ch, err)
			continue
		}

		total++
	}
	return total, nil
}

// dumpOneChannel dumps just one channel having ID = id.  If generateText is
// true, it will also generate a ID.txt text file.
func (app *App) dumpOneChannel(ctx context.Context, id string) error {
	f, err := os.Create(id + ".json")
	if err != nil {
		return err
	}
	defer f.Close()

	m, err := app.sd.DumpMessages(ctx, id)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		return err
	}
	if app.cfg.Output.IsText() {
		if err := app.saveText(m.ID+".txt", m); err != nil {
			dlog.Printf("error creating text file: %s", err)
		}
	}

	return nil
}

func (app *App) saveText(filename string, m *slackdump.Channel) error {
	dlog.Printf("generating %s", filename)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return m.ToText(app.sd, f)
}

// listEntities queries lists the supported entities, and writes the output to output.
func (app *App) listEntities(ctx context.Context) error {
	f, err := createFile(app.cfg.Output.Filename)
	if err != nil {
		return err
	}
	defer f.Close()

	dlog.Print("retrieving data...")
	rep, err := app.fetchEntity(ctx, app.cfg.ListFlags)
	if err != nil {
		return err
	}

	if err := app.formatEntity(f, rep, app.cfg.Output); err != nil {
		return err
	}
	return nil
}

// createFile creates the file, or opens the Stdout, if the filename is "-".
// It will return an error, if the things go pear-shaped.
func createFile(filename string) (f io.WriteCloser, err error) {
	if filename == "-" {
		f = os.Stdout
		return
	}
	return os.Create(filename)
}

// fetchEntity retrieves the data from the API according to the ListFlags.
func (app *App) fetchEntity(ctx context.Context, listFlags ListFlags) (rep slackdump.Reporter, err error) {
	switch {
	case listFlags.Channels:
		rep, err = app.sd.GetChannels(ctx)
		if err != nil {
			return
		}
	case listFlags.Users:
		rep = app.sd.Users
	default:
		err = errors.New("nothing to do")
	}
	return
}

// formatEntity formats the entity according to output specification.
func (app *App) formatEntity(w io.Writer, rep slackdump.Reporter, output Output) error {
	switch output.Format {
	case OutputTypeText:
		return rep.ToText(w)
	case OutputTypeJSON:
		enc := json.NewEncoder(w)
		return enc.Encode(rep)
	}
	return errors.New("invalid Output format")
}
