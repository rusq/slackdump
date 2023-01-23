package export

import (
	"context"
	"errors"
	"fmt"

	"github.com/rusq/dlog"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/export"
	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/rusq/slackdump/v2/internal/app/config"
	"github.com/rusq/slackdump/v2/internal/structures"
)

var CmdExport = &base.Command{
	Run:         nil,
	Wizard:      nil,
	UsageLine:   "slackdump export",
	Short:       "exports the Slack Workspace or individual conversations",
	Long:        ``, // TODO: add long description
	CustomFlags: false,
	PrintFlags:  true,
	RequireAuth: true,
}

var (
	options export.Options
)

func init() {
	// TODO: move TimeValue somewhere more appropriate once v1 is sunset.
	CmdExport.Flag.Var(ptr(config.TimeValue(options.Oldest)), "from", "timestamp of the oldest message to fetch")
	CmdExport.Flag.Var(ptr(config.TimeValue(options.Latest)), "to", "timestamp of the newest message to fetch")
	CmdExport.Flag.Var(&options.Type, "type", "export type")
	CmdExport.Flag.StringVar(&options.ExportToken, "export-token", "", "file export token to append to each of the file URLs")
}

func ptr[T any](a T) *T {
	return &a
}

func init() {
	CmdExport.Run = runExport
	CmdExport.Wizard = wizExport
}

func runExport(ctx context.Context, cmd *base.Command, args []string) error {
	if cfg.BaseLoc == "" {
		return errors.New("use -base to set the base output location")
	}
	var err error
	options.List, err = structures.NewEntityList(args)
	if err != nil {
		return fmt.Errorf("error parsing the entity list: %w", err)
	}

	prov, err := auth.FromContext(ctx)
	if err != nil {
		return err
	}

	fs, err := fsadapter.New(cfg.BaseLoc)
	if err != nil {
		return err
	}
	defer fs.Close()

	lg := dlog.FromContext(ctx)
	options.Logger = lg
	lg.Printf("initialised output location: %s", cfg.BaseLoc)

	sess, err := slackdump.NewWithOptions(ctx, prov, cfg.SlackOptions)
	if err != nil {
		return err
	}

	exp := export.New(sess, fs, options)
	return exp.Run(ctx)
}
