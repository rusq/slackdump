package diag

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/chunk"
)

var CmdRecord = &base.Command{
	UsageLine: "slackdump tools record",
	Short:     "chunk record commands",
	Commands:  []*base.Command{cmdRecordStream, cmdRecordState},
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

	sess, err := cfg.SlackdumpSession(ctx)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return err
	}

	var w io.Writer
	if *output == "" {
		w = os.Stdout
	} else {
		if f, err := os.Create(*output); err != nil {
			base.SetExitStatus(base.SApplicationError)
			return err
		} else {
			defer f.Close()
			w = f
		}
	}

	rec := chunk.NewRecorder(w)
	for _, ch := range args {
		cfg.Log.Printf("streaming channel %q", ch)
		if err := sess.Stream().SyncConversations(ctx, rec, ch); err != nil {
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
