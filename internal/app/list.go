package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/rusq/slackdump/v2/internal/structures"
)

// reporter is an interface defining output functions
type reporter interface {
	ToText(w io.Writer, ui structures.UserIndex) error
}

// runListEntities queries lists the supported entities, and writes the output to
// the output defined in the app.cfg.
func (app *App) listEntities(ctx context.Context, output Output, lf ListFlags) error {
	f, err := createFile(output.Filename)
	if err != nil {
		return err
	}
	defer f.Close()

	app.l().Print("retrieving data...")
	rep, err := app.fetchEntity(ctx, lf)
	if err != nil {
		return err
	}

	if err := app.formatEntity(f, rep, output); err != nil {
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
func (app *App) fetchEntity(ctx context.Context, listFlags ListFlags) (rep reporter, err error) {
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

// formatEntity formats reporter output as defined in the "Output".
func (app *App) formatEntity(w io.Writer, rep reporter, output Output) error {
	switch output.Format {
	case OutputTypeText:
		return rep.ToText(w, app.sd.UserIndex)
	case OutputTypeJSON:
		enc := json.NewEncoder(w)
		return enc.Encode(rep)
	}
	return errors.New("invalid Output format")
}
