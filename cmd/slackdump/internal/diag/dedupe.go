package diag

import (
	"context"
	"fmt"
	"os"

	"github.com/jmoiron/sqlx"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	dedupecmd "github.com/rusq/slackdump/v4/cmd/slackdump/internal/diag/dedupe"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/source"
)

var cmdDedupe = &base.Command{
	UsageLine:  "slackdump tools dedupe [flags] <archive_directory>",
	Short:      "deduplicate overlap entities from resume runs",
	FlagMask:   cfg.OmitAll,
	PrintFlags: true,
	Long: `
Dedupe removes identical duplicate messages, users, channels, channel users,
and files created by resume look-back overlap. The latest copy of each
identical payload is kept. By default it only reports what would be removed.
Pass the archive directory, not the slackdump.sqlite file. Use -execute to
perform deduplication.
`,
}

var dedupeFlags struct {
	execute bool
}

func init() {
	cmdDedupe.Run = runDedupe
	cmdDedupe.Flag.BoolVar(&dedupeFlags.execute, "execute", false, "actually remove duplicate entities")
}

func ensureDb(ctx context.Context, dir string) (*sqlx.DB, error) {
	src, err := source.Load(ctx, dir)
	if err != nil {
		base.SetExitStatus(base.SInvalidParameters)
		return nil, err
	}
	defer src.Close()
	if !src.Type().Has(source.FDatabase) {
		base.SetExitStatus(base.SInvalidParameters)
		return nil, fmt.Errorf("source type %q does not contain a database archive, use 'slackdump convert -f database' to convert it", src.Type())
	}

	conn, err := bootstrap.Database(dir)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return nil, fmt.Errorf("error opening database: %w", err)
	}
	return conn, nil

}

func runDedupe(ctx context.Context, cmd *base.Command, args []string) error {
	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	if cmd.Flag.NArg() != 1 {
		cmd.Flag.Usage()
		return nil
	}

	dir := cmd.Flag.Arg(0)
	conn, err := ensureDb(ctx, dir)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = dedupecmd.Run(ctx, conn, dedupecmd.Options{
		Execute:  dedupeFlags.execute,
		Report:   os.Stdout,
		Database: dir,
	})
	return err
}
