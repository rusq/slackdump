package diag

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/internal/processors"
)

var CmdRecord = &base.Command{
	UsageLine: "slackdump diag record",
	Short:     "event record commands",
	Commands:  []*base.Command{CmdRecordStream, CmdRecordState},
}

var CmdRecordStream = &base.Command{
	UsageLine:   "slackdump diag record stream [options] <channel>",
	Short:       "dump slack data in a event record format",
	FlagMask:    cfg.OmitBaseLocFlag | cfg.OmitDownloadFlag,
	PrintFlags:  true,
	RequireAuth: true,
}

var CmdRecordState = &base.Command{
	UsageLine:   "slackdump diag record state [options] <record_file.jsonl>",
	Short:       "print state of the record",
	FlagMask:    cfg.OmitAll,
	PrintFlags:  true,
	RequireAuth: false,
}

func init() {
	// break init cycle
	CmdRecordStream.Run = runRecord
}

var output = CmdRecord.Flag.String("output", "", "output file")

func runRecord(ctx context.Context, _ *base.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("missing channel argument")
	}

	prov, err := auth.FromContext(ctx)
	if err != nil {
		return err
	}
	sess, err := slackdump.New(ctx, prov, cfg.SlackConfig)
	if err != nil {
		return err
	}
	defer sess.Close()

	var w io.Writer
	if *output == "" {
		w = os.Stdout
	} else {
		if f, err := os.Create(*output); err != nil {
			return err
		} else {
			defer f.Close()
			w = f
		}
	}

	rec := processors.NewRecorder(w)
	for _, ch := range args {
		cfg.Log.Printf("streaming channel %q", ch)
		if err := sess.Stream(ctx, ch, rec, time.Time{}, time.Time{}); err != nil {
			if err2 := rec.Close(); err2 != nil {
				return fmt.Errorf("error streaming channel %q: %w; error closing recorder: %v", ch, err, err2)
			}
			return err
		}
	}
	if err := rec.Close(); err != nil {
		return err
	}
	st, err := rec.State()
	if err != nil {
		return err
	}
	return st.Save(*output + ".state")
}

func init() {
	// break init cycle
	CmdRecordState.Run = runRecordState
}

func runRecordState(ctx context.Context, _ *base.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("missing record file argument")
	}
	f, err := os.Open(args[0])
	if err != nil {
		return err
	}
	defer f.Close()

	pl, err := processors.NewPlayer(f)
	if err != nil {
		return err
	}
	state, err := pl.State()
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(state)
}
