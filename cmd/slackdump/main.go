package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"runtime/trace"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/joho/godotenv"

	_ "modernc.org/sqlite"
	// _ "github.com/mattn/go-sqlite3"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/apiconfig"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/archive"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/convertcmd"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/diag"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/dump"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/emoji"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/export"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/format"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/help"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/list"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/man"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/resume"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/view"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/wizard"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/workspace"
	"github.com/rusq/slackdump/v3/internal/osext"
)

func init() {
	base.Slackdump.Commands = []*base.Command{
		workspace.CmdWorkspace,
		archive.CmdArchive,
		export.CmdExport,
		dump.CmdDump,
		archive.CmdSearch,
		resume.CmdResume,
		convertcmd.CmdConvert,
		list.CmdList,
		emoji.CmdEmoji,
		diag.CmdDiag,
		apiconfig.CmdConfig,
		format.CmdFormat,
		view.CmdView,
		wizard.CmdWizard,
		CmdVersion,

		man.Quickstart,
		man.WhatsNew,
		man.Syntax,
		man.Login,
		man.Chunk,
		man.Migration,
		man.Transfer,
		man.Troubleshooting,
	}
}

func main() {
	if osext.IsRoot() {
		slog.Warn("slackdump:  courageously running as root, hope you know what you're doing")
	}

	flag.Usage = base.Usage
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		if !osext.IsInteractive() {
			base.Usage()
			// Usage terminates the program.
		}

		next, err := whatDo()
		if err != nil {
			log.Fatal(err)
		}
		switch next {
		case choiceExit:
			return
		case choiceWizard:
			args = append(args, "wiz")
		case choiceHelp:
			fallthrough
		default:
			base.Usage()
		}
	}
	base.CmdName = args[0]
	if args[0] == "help" {
		help.Help(os.Stdout, args[1:])
		return
	}

BigCmdLoop:
	for bigCmd := base.Slackdump; ; {
		for _, cmd := range bigCmd.Commands {
			if cmd.Name() != args[0] {
				continue
			}
			if len(cmd.Commands) > 0 {
				bigCmd = cmd
				args = args[1:]
				if len(args) == 0 {
					help.PrintUsage(os.Stderr, bigCmd)
					base.SetExitStatus(base.SHelpRequested)
					base.Exit()
				}
				if args[0] == "help" {
					help.Help(os.Stdout, append(strings.Split(base.CmdName, " "), args[1:]...))
					return
				}
				base.CmdName += " " + args[0]
				continue BigCmdLoop
			}
			if !cmd.Runnable() {
				continue
			}
			if err := invoke(cmd, args); err != nil {
				if errors.Is(err, context.Canceled) {
					slog.Info("operation cancelled")
				} else {
					msg := fmt.Sprintf("%03[1]d (%[1]s): %[2]s.", base.ExitStatus(), err)
					slog.Error(msg)
				}
			}
			base.Exit()
			return
		}
		helpArg := ""
		if i := strings.LastIndex(base.CmdName, " "); i >= 0 {
			helpArg = " " + base.CmdName[:i]
		}
		fmt.Fprintf(os.Stderr, "slackdump %s: unknown command\nRun 'slackdump help%s' for usage.\n", base.CmdName, helpArg)
		base.SetExitStatus(base.SInvalidParameters)
		base.Exit()
	}
}

func init() {
	base.Usage = mainUsage
}

func mainUsage() {
	help.PrintUsage(os.Stderr, base.Slackdump)
	os.Exit(2)
}

func invoke(cmd *base.Command, args []string) error {
	if cmd.CustomFlags {
		args = args[1:]
	} else {
		var err error
		args, err = parseFlags(cmd, args)
		if err != nil {
			return err
		}
	}
	if cfg.LoadSecrets {
		// load secrets only if we're told to.
		loadSecrets(secretFiles)
	} else {
		if os.Getenv("SLACK_TOKEN") != "" {
			log.Println("warning: SLACK_TOKEN is set in the environment, but not used, run with -env flag to use it")
		}
	}

	// maybe start trace
	traceStop := initTrace(cfg.TraceFile)
	base.AtExit(traceStop)
	initDebug()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	ctx, task := trace.NewTask(ctx, "command")
	defer task.End()

	// initialise default logging.
	if lg, err := initLog(cfg.LogFile, cfg.JsonHandler, cfg.Verbose); err != nil {
		base.SetExitStatus(base.SInitializationError)
		return err
	} else {
		lg.With("command", cmd.Name())
		cfg.Log = lg
	}

	if cmd.RequireAuth {
		trace.Logf(ctx, "invoke", "command %s requires auth", cmd.Name())
		var err error
		ctx, err = workspace.CurrentOrNewProviderCtx(ctx)
		if err != nil {
			base.SetExitStatus(base.SAuthError)
			return fmt.Errorf("auth error: %w", err)
		}
	}
	trace.Log(ctx, "command", fmt.Sprint("Running ", cmd.Name(), " command"))
	return cmd.Run(ctx, cmd, args)
}

func parseFlags(cmd *base.Command, args []string) ([]string, error) {
	cfg.SetBaseFlags(&cmd.Flag, cmd.FlagMask)
	cmd.Flag.Usage = func() { cmd.Usage() }
	if err := cmd.Flag.Parse(args[1:]); err != nil {
		return nil, err
	}
	if cfg.ConfigFile == "" {
		return cmd.Flag.Args(), nil
	}

	// load the API limit configuration file.
	limits, err := apiconfig.Load(cfg.ConfigFile)
	if err != nil {
		return nil, err
	}
	if err := cfg.Limits.Apply(limits); err != nil {
		return nil, err
	}
	return cmd.Flag.Args(), nil
}

func iftrue[T any](cond bool, t T, f T) T {
	if cond {
		return t
	}
	return f
}

// secrets defines the names of the supported secret files that we load our
// secrets from.  Inexperienced Windows users might have bad experience trying
// to create .env file with the notepad as it will battle for having the
// "txt" extension.  Let it have it.
var secretFiles = []string{".env", ".env.txt", "secrets.txt"}

// loadSecrets load secrets from the files in secrets slice.
func loadSecrets(files []string) {
	for _, f := range files {
		_ = godotenv.Load(f)
	}
}

type choice string

const (
	choiceUnknown choice = ""
	choiceHelp    choice = "Print help and exit"
	choiceWizard  choice = "Run wizard"
	choiceExit    choice = "Exit"
)

func whatDo() (choice, error) {
	versionstr := cfg.Version.String()

	var ans choice
	err := huh.NewForm(huh.NewGroup(huh.NewSelect[choice]().
		Title(versionstr).
		Description("What would you like to do?").
		Options(
			huh.NewOption(string(choiceHelp), choiceHelp),
			huh.NewOption(string(choiceWizard), choiceWizard),
			huh.NewOption(string(choiceExit), choiceExit),
		).Value(&ans))).WithTheme(ui.HuhTheme()).WithKeyMap(ui.DefaultHuhKeymap).Run()

	return ans, err
}
