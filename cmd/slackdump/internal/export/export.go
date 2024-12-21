package export

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/chunk/transform/fileproc"
	"github.com/rusq/slackdump/v3/internal/structures"
)

var CmdExport = &base.Command{
	Run:         nil,
	Wizard:      nil,
	UsageLine:   "slackdump export",
	Short:       "exports the Slack Workspace or individual conversations",
	FlagMask:    cfg.OmitUserCacheFlag,
	Long:        ``, // TODO: add long description
	CustomFlags: false,
	PrintFlags:  true,
	RequireAuth: true,
}

type exportFlags struct {
	ExportStorageType fileproc.StorageType
	ExportToken       string
}

var (
	compat  bool
	options = exportFlags{
		ExportStorageType: fileproc.STmattermost,
	}
)

func init() {
	CmdExport.Flag.Var(&options.ExportStorageType, "type", "export file storage type")
	CmdExport.Flag.StringVar(&options.ExportToken, "export-token", "", "file export token to append to each of the file URLs")

	CmdExport.Run = runExport
	CmdExport.Wizard = wizExport
}

func runExport(ctx context.Context, cmd *base.Command, args []string) error {
	start := time.Now()
	if strings.TrimSpace(cfg.Output) == "" {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("use -base to set the base output location")
	}
	if !cfg.DownloadFiles {
		options.ExportStorageType = fileproc.STnone
	}
	list, err := structures.NewEntityList(args)
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return fmt.Errorf("error parsing the entity list: %w", err)
	}

	sess, err := bootstrap.SlackdumpSession(ctx)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}

	fsa, err := fsadapter.New(cfg.Output)
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return err
	}
	lg := cfg.Log
	defer func() {
		lg.DebugContext(ctx, "closing the fsadapter")
		fsa.Close()
	}()

	if err := export(ctx, sess, fsa, list, options); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("export failed: %w", err)
	}

	lg.InfoContext(ctx, "export completed", "took", time.Since(start).String())
	return nil
}
