package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/types"
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
	if err := input.producer(func(channelID string) error {
		if err := app.dumpOne(ctx, channelID, app.newDumpFunc(channelID)); err != nil {
			app.l().Printf("error processing: %q (conversation will be skipped): %s", channelID, err)
			return errSkip
		}
		total++
		return nil
	}); err != nil {
		return total, err
	}
	return total, nil
}

type dumpFunc func(context.Context, string, time.Time, time.Time, ...slackdump.ProcessFunc) (*types.Conversation, error)

// newDumpFunc returns the appropriate dump function depending on the input s.
func (app *App) newDumpFunc(s string) dumpFunc {
	if strings.HasPrefix(strings.ToLower(s), "https://") {
		return app.sd.DumpURL
	} else {
		return app.sd.DumpMessages
	}
}

// renderFilename returns the filename that is rendered according to the
// file naming template.
func (app *App) renderFilename(c *types.Conversation) string {
	var buf strings.Builder
	if err := app.tmpl.ExecuteTemplate(&buf, filenameTmplName, c); err != nil {
		// this should nevar happen
		panic(err)
	}
	return buf.String()
}

// dumpOneChannel dumps just one channel specified by channelInput.  If
// generateText is true, it will also generate a ID.txt text file.
func (app *App) dumpOne(ctx context.Context, channelInput string, fn dumpFunc) error {
	cnv, err := fn(ctx, channelInput, time.Time(app.cfg.Oldest), time.Time(app.cfg.Latest))
	if err != nil {
		return err
	}

	return app.writeFiles(app.renderFilename(cnv), cnv)
}

// writeFiles writes the conversation to disk.  If text output is set, it will
// also generate a text file having the same name as JSON file.
func (app *App) writeFiles(name string, cnv *types.Conversation) error {
	if err := app.writeJSON(name+".json", cnv); err != nil {
		return err
	}
	if app.cfg.Output.IsText() {
		if err := app.writeText(name+".txt", cnv); err != nil {
			return err
		}
	}
	return nil
}

func (app *App) writeJSON(filename string, m any) error {
	app.l().Printf("generating %s", filename)
	f, err := app.fs.Create(filename)
	if err != nil {
		return fmt.Errorf("error writing %q: %w", filename, err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		return fmt.Errorf("error encoding %q: %w", filename, err)
	}
	return nil
}

func (app *App) writeText(filename string, m *types.Conversation) error {
	app.l().Printf("generating %s", filename)
	f, err := app.fs.Create(filename)
	if err != nil {
		return fmt.Errorf("error writing %q: %w", filename, err)
	}
	defer f.Close()

	return m.ToText(f, app.sd.UserIndex)
}
