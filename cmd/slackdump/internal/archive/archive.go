package archive

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/control"
	"github.com/rusq/slackdump/v3/internal/chunk/transform/fileproc"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/logger"
	"github.com/rusq/slackdump/v3/stream"
)

//go:embed assets/archive.md
var mdArchive string

var CmdArchive = &base.Command{
	Run:         RunArchive,
	UsageLine:   "slackdump archive [flags] [link1[ link 2[ link N]]]",
	Short:       "archive the workspace or individual conversations on disk",
	Long:        mdArchive,
	FlagMask:    cfg.OmitUserCacheFlag | cfg.OmitCacheDir,
	Wizard:      archiveWizard,
	RequireAuth: true,
	PrintFlags:  true,
}

const zipExt = ".ZIP"

// StripZipExt removes the .zip extension from the string.
func StripZipExt(s string) string {
	if strings.HasSuffix(strings.ToUpper(s), zipExt) {
		return s[:len(s)-len(zipExt)]
	}
	return s
}

var (
	errNoOutput = errors.New("output directory is required")
)

func RunArchive(ctx context.Context, cmd *base.Command, args []string) error {
	list, err := structures.NewEntityList(args)
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return err
	}

	cfg.Output = StripZipExt(cfg.Output)
	if cfg.Output == "" {
		base.SetExitStatus(base.SInvalidParameters)
		return errNoOutput
	}

	cd, err := chunk.CreateDir(cfg.Output)
	if err != nil {
		base.SetExitStatus(base.SGenericError)
		return err
	}

	sess, err := bootstrap.SlackdumpSession(ctx)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return err
	}
	lg := logger.FromContext(ctx)
	stream := sess.Stream(
		stream.OptLatest(time.Time(cfg.Latest)),
		stream.OptOldest(time.Time(cfg.Oldest)),
		stream.OptResultFn(resultLogger(lg)),
	)
	dl, stop := fileproc.NewDownloader(
		ctx,
		cfg.DownloadFiles,
		sess.Client(),
		fsadapter.NewDirectory(cd.Name()),
		lg,
	)
	defer stop()
	// we are using the same file subprocessor as the mattermost export.
	subproc := fileproc.NewExport(fileproc.STmattermost, dl)
	ctrl := control.New(cd, stream, control.WithLogger(lg), control.WithFiler(subproc))
	if err := ctrl.Run(ctx, list); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	lg.Printf("Recorded workspace data to %s", cd.Name())

	return nil
}

func resultLogger(lg logger.Interface) func(sr stream.Result) error {
	return func(sr stream.Result) error {
		lg.Printf("%s", sr)
		return nil
	}
}

func archiveWizard(ctx context.Context, cmd *base.Command, args []string) error {
	selected := "dates"
LOOP:
	for {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Key("selection").
					Title("Select the workspace or conversation to archive").
					Description("Select the workspace or conversation to archive").
					Options(
						huh.NewOption("Specify date range", "dates"),
						huh.NewOption("Custom API limits config file", "config"),
						huh.NewOption(fmt.Sprintf("Enterprise mode enabled? (%v)", cfg.ForceEnterprise), "enterprise"),
						huh.NewOption(fmt.Sprintf("Export files? (%v)", cfg.DownloadFiles), "files"),
						huh.NewOption(fmt.Sprintf("Output directory:  %q", StripZipExt(cfg.Output)), "output"),
						huh.NewOption("Run!", "run"),
						huh.NewOption(strings.Repeat("-", 10), ""),
						huh.NewOption("Exit archive wizard", "exit"),
					).Value(&selected).
					DescriptionFunc(func() string {
						switch selected {
						case "dates":
							return "Specify the date range for the archive"
						case "config":
							return "Specify the custom API limits configuration file"
						case "enterprise":
							return "Enable or disable enterprise mode"
						case "files":
							return "Enable or disable files download"
						case "output":
							return "Specify the output directory, or use the default one"
						case "run":
							return "Run the archive"
						case "exit":
							return "Exit the archive wizard"
						default:
							return ""
						}
					}, &selected).
					WithTheme(cfg.Theme),
			),
		)
		if err := form.Run(); err != nil {
			return err
		}
		switch selected {
		case "exit":
			break LOOP
		case "enterprise":
			cfg.ForceEnterprise = !cfg.ForceEnterprise
		case "files":
			cfg.DownloadFiles = !cfg.DownloadFiles
		}
	}

	return nil
}
