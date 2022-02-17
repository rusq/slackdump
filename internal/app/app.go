package app

import (
	"context"
	"encoding/json"
	"errors"
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
	sd, err := slackdump.NewWithOptions(
		ctx,
		app.cfg.Creds.Token,
		app.cfg.Creds.Cookie,
		app.cfg.Options,
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
	} else {
		n, err := app.dump(ctx, app.cfg.Input)
		if err != nil {
			return err
		}
		dlog.Printf("job finished, dumped %d channels", n)
	}
	return nil
}

// dump dumps the input, if dumpfiles is true, it will save the files into a
// respective directory with ID of the channel as the name.  If generateText is
// true, it will additionally format the conversation as text file and write it
// to <ID>.txt file.
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
func (app *App) dump(ctx context.Context, input Input) (int, error) {
	if !input.IsValid() {
		return 0, errors.New("no valid input")
	}

	total := 0
	if err := input.producer(func(s string) error {
		if err := app.dumpOne(ctx, s, app.newDumpFunc(s)); err != nil {
			dlog.Printf("error processing: %q (conversation will be skipped): %s", s, err)
			return errSkip
		}
		total++
		return nil
	}); err != nil {
		return total, err
	}
	return total, nil
}

func (app *App) newDumpFunc(s string) dumpFunc {
	if strings.HasPrefix(strings.ToLower(s), "https://") {
		return app.sd.DumpURL
	} else {
		return app.sd.DumpMessages
	}
}

type dumpFunc func(context.Context, string) (*slackdump.Conversation, error)

// dumpOneChannel dumps just one channel having ID = id.  If generateText is
// true, it will also generate a ID.txt text file.
func (app *App) dumpOne(ctx context.Context, s string, fn dumpFunc) error {
	cnv, err := fn(ctx, s)
	if err != nil {
		return err
	}

	return app.writeFile(cnv.String(), cnv)
}

func (app *App) writeFile(name string, cnv *slackdump.Conversation) error {
	f, err := os.Create(name + ".json")
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cnv); err != nil {
		return err
	}
	if app.cfg.Output.IsText() {
		if err := app.saveText(name+".txt", cnv); err != nil {
			dlog.Printf("error creating text file: %s", err)
		}
	}

	return nil
}

func (app *App) saveText(filename string, m *slackdump.Conversation) error {
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
// It will return an error, if things go pear-shaped.
func createFile(filename string) (f io.WriteCloser, err error) {
	if filename == "-" {
		f = os.Stdout
		return
	}
	return os.Create(filename)
}

// openFile opens the file, or opens the Stdin, if the filename is "-".
// It will return an error, if shit happens.
func openFile(filename string) (f io.ReadCloser, err error) {
	if filename == "-" {
		f = os.Stdin
		return
	}
	return os.Open(filename)
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
		rep, err = app.sd.GetUsers(ctx)
		if err != nil {
			return
		}
	default:
		err = errors.New("nothing to do")
	}
	return
}

// formatEntity formats the entity according to output specification.
func (app *App) formatEntity(w io.Writer, rep slackdump.Reporter, output Output) error {
	switch output.Format {
	case OutputTypeText:
		return rep.ToText(app.sd, w)
	case OutputTypeJSON:
		enc := json.NewEncoder(w)
		return enc.Encode(rep)
	}
	return errors.New("invalid Output format")
}
