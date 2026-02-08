// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
package diag

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/internal/chunk"
	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase"
	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase/repository"
	"github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/stream"
)

var cmdRecord = &base.Command{
	UsageLine:  "slackdump tools record",
	Short:      "chunk record commands",
	Commands:   []*base.Command{cmdRecordStream},
	HideWizard: true,
}

var cmdRecordStream = &base.Command{
	UsageLine: "slackdump tools record stream [options] <channel>",
	Short:     "dump slack data in a chunk record format",
	Long: `
# Record tool

Records the data from a channel in a chunk record format.

See also: slackdump tool obfuscate
`,
	FlagMask:    cfg.OmitOutputFlag | cfg.OmitWithFilesFlag,
	PrintFlags:  true,
	RequireAuth: true,
}

func init() {
	// break init cycle
	cmdRecordStream.Run = runRecord
}

// var output = cmdRecordStream.Flag.String("output", "", "output file")

func runRecord(ctx context.Context, _ *base.Command, args []string) error {
	if len(args) == 0 {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("missing channel argument")
	}

	client, err := bootstrap.Slack(ctx)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return err
	}

	// var w io.Writer
	// if *output == "" {
	// 	w = os.Stdout
	// } else {
	// 	if f, err := os.Create(*output); err != nil {
	// 		base.SetExitStatus(base.SApplicationError)
	// 		return err
	// 	} else {
	// 		defer f.Close()
	// 		w = f
	// 	}
	// }

	db, err := sqlx.Open(repository.Driver, "record.db")
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	defer db.Close()

	runParams := dbase.SessionInfo{
		FromTS:         (*time.Time)(&cfg.Oldest),
		ToTS:           (*time.Time)(&cfg.Latest),
		FilesEnabled:   cfg.WithFiles,
		AvatarsEnabled: cfg.WithAvatars,
		Mode:           "record",
		Args:           strings.Join(os.Args, "|"),
	}

	p, err := dbase.New(ctx, db, runParams)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	defer p.Close()

	// rec := chunk.NewRecorder(w)
	rec := chunk.NewCustomRecorder(p)
	for _, ch := range args {
		lg := cfg.Log.With("channel_id", ch)
		lg.InfoContext(ctx, "streaming")
		if err := stream.New(client, cfg.Limits).SyncConversations(ctx, rec, structures.EntityItem{Id: ch}); err != nil {
			if err2 := rec.Close(); err2 != nil {
				base.SetExitStatus(base.SApplicationError)
				return fmt.Errorf("error streaming channel %q: %w; error closing recorder: %v", ch, err, err2)
			}
			return err
		}
	}
	if err := rec.Close(); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	return nil
}
