package diag

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/dbproc"
	"github.com/rusq/slackdump/v3/internal/structures"
)

var cmdRecord = &base.Command{
	UsageLine:  "slackdump tools record",
	Short:      "chunk record commands",
	Commands:   []*base.Command{cmdRecordStream, cmdRecordState},
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
	FlagMask:    cfg.OmitOutputFlag | cfg.OmitDownloadFlag,
	PrintFlags:  true,
	RequireAuth: true,
}

var cmdRecordState = &base.Command{
	UsageLine:   "slackdump tools record state [options] <record_file.jsonl>",
	Short:       "print state of the record",
	FlagMask:    cfg.OmitAll,
	PrintFlags:  true,
	RequireAuth: false,
}

func init() {
	// break init cycle
	cmdRecordStream.Run = runRecord
}

var output = cmdRecordStream.Flag.String("output", "", "output file")

func runRecord(ctx context.Context, _ *base.Command, args []string) error {
	if len(args) == 0 {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("missing channel argument")
	}

	sess, err := bootstrap.SlackdumpSession(ctx)
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

	db, err := sqlx.Open("sqlite", "record.db")
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	defer db.Close()

	runParams := dbproc.Parameters{
		FromTS:         (*time.Time)(&cfg.Oldest),
		ToTS:           (*time.Time)(&cfg.Latest),
		FilesEnabled:   cfg.DownloadFiles,
		AvatarsEnabled: cfg.DownloadAvatars,
		Mode:           "record",
		Args:           strings.Join(os.Args, "|"),
	}

	p, err := dbproc.New(ctx, db, runParams)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	defer p.Close()

	// rec := chunk.NewRecorder(w)
	rec := chunk.NewCustomRecorder("record", p)
	for _, ch := range args {
		lg := cfg.Log.With("channel_id", ch)
		lg.InfoContext(ctx, "streaming")
		if err := sess.Stream().SyncConversations(ctx, rec, structures.EntityItem{Id: ch}); err != nil {
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
	st, err := rec.State()
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	if err := st.Save(*output + ".state"); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	return nil
}

func init() {
	// break init cycle
	cmdRecordState.Run = runRecordState
}

func runRecordState(ctx context.Context, _ *base.Command, args []string) error {
	if len(args) == 0 {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("missing record file argument")
	}
	f, err := os.Open(args[0])
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	defer f.Close()

	cf, err := chunk.FromReader(f)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	state, err := cf.State()
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(state); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	return nil
}
