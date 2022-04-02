package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/rusq/dlog"
	"github.com/rusq/slackdump"
)

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

type dumpFunc func(context.Context, string, time.Time, time.Time) (*slackdump.Conversation, error)

func (app *App) newDumpFunc(s string) dumpFunc {
	if strings.HasPrefix(strings.ToLower(s), "https://") {
		return app.sd.DumpURLInTimeframe
	} else {
		return app.sd.DumpMessagesInTimeframe
	}
}

func (app *App) renderFilename(c *slackdump.Conversation) string {
	var buf strings.Builder
	if err := app.tmpl.ExecuteTemplate(&buf, filenameTmplName, c); err != nil {
		// this should nevar happen
		panic(err)
	}
	return buf.String()
}

// dumpOneChannel dumps just one channel having ID = id.  If generateText is
// true, it will also generate a ID.txt text file.
func (app *App) dumpOne(ctx context.Context, s string, fn dumpFunc) error {
	cnv, err := fn(ctx, s, time.Time(app.cfg.Oldest), time.Time(app.cfg.Latest))
	if err != nil {
		return err
	}

	return app.writeFile(app.renderFilename(cnv), cnv)
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
