package diag

import (
	"context"
	"errors"
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
	UsageLine:   "slackdump diag record [options] <channel>",
	Short:       "dump slack data in a event record format",
	FlagMask:    cfg.OmitBaseLocFlag | cfg.OmitDownloadFlag,
	PrintFlags:  true,
	RequireAuth: true,
}

func init() {
	// break init cycle
	CmdRecord.Run = runRecord
}

var output = CmdRecord.Flag.String("output", "", "output file")

func runRecord(ctx context.Context, cmd *base.Command, args []string) error {
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
		f, err := os.Create("output.jsonl")
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}

	rec := processors.NewRecorder(w)
	defer rec.Close()

	if err := sess.Stream(ctx, args[0], rec, time.Time{}, time.Time{}); err != nil {
		return err
	}
	return nil
}
