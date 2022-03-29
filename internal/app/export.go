package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rusq/slackdump"
	"github.com/slack-go/slack"
)

// Export performs the full export of slack workspace in slack export compatible
// format.
func (app *App) Export(ctx context.Context, dir string) error {
	if dir == "" { // dir is passed from app.cfg.ExportDirectory
		return errors.New("export directory not specified")
	}

	export := newSlackExport(dir, app.sd)

	// export users to users.json
	if err := export.users(ctx); err != nil {
		return err
	}

	// export channels to channels.json
	if err := export.channels(ctx); err != nil {
		return err
	}

	return nil
}

type slackExport struct {
	dir string

	dumper *slackdump.SlackDumper
}

func newSlackExport(dir string, dumper *slackdump.SlackDumper) *slackExport {
	return &slackExport{dir: dir, dumper: dumper}
}

func (se *slackExport) users(ctx context.Context) error {
	// fetch users and save them.
	users, err := se.dumper.GetUsers(ctx)
	if err != nil {
		return err
	}

	return se.serialize(filepath.Join(se.dir, "users.json"), users)
}

func (se *slackExport) channels(ctx context.Context) error {
	if err := se.dumper.StreamChannels(ctx, slackdump.AllChanTypes, func(ch slack.Channel) error {
		// TODO
		return nil
	}); err != nil {
		return fmt.Errorf("channels: error: %w", err)
	}

	return nil
}

// serialize writes the data in json format to filename.
func (*slackExport) serialize(filename string, data interface{}) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("serialize: failed to create %q: %w", filename, err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")

	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("serialize: failed to write %q: %w", filename, err)
	}

	return nil
}
