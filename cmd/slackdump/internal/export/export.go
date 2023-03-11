package export

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rusq/dlog"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/export"
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
	options = export.Config{
		Type:   export.TStandard,
		Oldest: time.Time(cfg.Oldest),
		Latest: time.Time(cfg.Latest),
	}
)

func init() {
	// TODO: move TimeValue somewhere more appropriate once v1 is sunset.
	CmdExport.Flag.Var(&options.Type, "type", "export type")
	CmdExport.Flag.StringVar(&options.ExportToken, "export-token", "", "file export token to append to each of the file URLs")
}

func init() {
	CmdExport.Run = runExport
	CmdExport.Wizard = wizExport
}

func runExport(ctx context.Context, cmd *base.Command, args []string) error {
	if cfg.SlackConfig.BaseLocation == "" {
		return errors.New("use -base to set the base output location")
	}
	list, err := structures.NewEntityList(args)
	if err != nil {
		return fmt.Errorf("error parsing the entity list: %w", err)
	}

	prov, err := auth.FromContext(ctx)
	if err != nil {
		return err
	}

	return exportV2(ctx, prov, list)
}

func exportV2(ctx context.Context, prov auth.Provider, list *structures.EntityList) error {
	options.List = list
	options.Logger = dlog.FromContext(ctx)
	sess, err := slackdump.New(ctx, prov, cfg.SlackConfig)
	if err != nil {
		return err
	}
	defer sess.Close()

	exp := export.New(sess, options)
	return exp.Run(ctx)
}

// func exportV3(ctx context.Context, prov auth.Provider, list *structures.EntityList) error {

// }
