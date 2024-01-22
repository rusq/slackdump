package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/trace"
	"strings"
	"syscall"

	"github.com/charmbracelet/huh"
	"github.com/joho/godotenv"
	"github.com/rusq/dlog"
	"github.com/rusq/tracer"
	"golang.org/x/term"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/apiconfig"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/authcmd"
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
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/record"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/wizard"
	"github.com/rusq/slackdump/v3/logger"
)

func init() {
	loadSecrets(secretFiles)

	base.Slackdump.Commands = []*base.Command{
		wizard.CmdWizard,
		export.CmdExport,
		dump.CmdDump,
		record.CmdRecord,
		convertcmd.CmdConvert,
		list.CmdList,
		emoji.CmdEmoji,
		authcmd.CmdWorkspace,
		diag.CmdDiag,
		apiconfig.CmdConfig,
		format.CmdFormat,
		CmdVersion,

		man.WhatsNew,
		man.Login,
		man.Chunk,
	}
}

func main() {
	if isRoot() {
		dlog.Fatal("slackdump:  cowardly refusing to run as root")
	}

	flag.Usage = base.Usage
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		if !isInteractive() {
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
				dlog.Printf("Error %03[1]d (%[1]s): %[2]s", base.ExitStatus(), err)
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

	// maybe start trace
	if err := initTrace(cfg.TraceFile); err != nil {
		base.SetExitStatus(base.SGenericError)
		return fmt.Errorf("failed to start trace: %s", err)
	}

	ctx, task := trace.NewTask(context.Background(), "command")
	defer task.End()

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	// initialise default logging.
	if lg, err := initLog(cfg.LogFile, cfg.Verbose); err != nil {
		return err
	} else {
		lg.SetPrefix(cmd.Name() + ": ")
		ctx = logger.NewContext(ctx, lg)
		cfg.Log = lg
	}

	if cmd.RequireAuth {
		trace.Logf(ctx, "invoke", "command %s requires auth", cmd.Name())
		var err error
		ctx, err = authcmd.AuthCurrentCtx(ctx, cfg.CacheDir(), cfg.Workspace)
		if err != nil {
			trace.Logf(ctx, "invoke", "auth error: %s", err)
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

// initTrace initialises the tracing.  If the filename is not empty, the file
// will be opened, trace will write to that file.  Returns the stop function
// that must be called in the deferred call.  If the error is returned the stop
// function is nil.
func initTrace(filename string) error {
	if filename == "" {
		return nil
	}

	dlog.Printf("trace will be written to %q", filename)

	trc := tracer.New(filename)
	if err := trc.Start(); err != nil {
		return nil
	}

	stop := func() {
		if err := trc.End(); err != nil {
			dlog.Printf("failed to write the trace file: %s", err)
		}
	}
	base.AtExit(stop)
	return nil
}

// initLog initialises the logging and returns the context with the Logger. If
// the filename is not empty, the file will be opened, and the logger output will
// be switch to that file. Returns the initialised logger, stop function and an
// error, if any. The stop function must be called in the deferred call, it will
// close the log file, if it is open. If the error is returned the stop function
// is nil.
func initLog(filename string, verbose bool) (*dlog.Logger, error) {
	lg := logger.Default
	lg.SetDebug(verbose)

	if filename == "" {
		return lg, nil
	}

	lg.Debugf("log messages will be written to: %q", filename)
	lf, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return lg, fmt.Errorf("failed to create the log file: %w", err)
	}
	lg.SetOutput(lf)

	base.AtExit(func() {
		if err := lf.Close(); err != nil {
			dlog.Printf("failed to close the log file: %s", err)
		}
	})

	return lg, nil
}

// secrets defines the names of the supported secret files that we load our
// secrets from.  Inexperienced windows users might have bad experience trying
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
	fmt.Println()
	printVersion()
	fmt.Println()

	var ans choice
	err := huh.NewSelect[choice]().
		Title("What do you want to do?").
		Options(
			huh.NewOption(string(choiceHelp), choiceHelp),
			huh.NewOption(string(choiceWizard), choiceWizard),
			huh.NewOption(string(choiceExit), choiceExit),
		).Value(&ans).Run()

	return ans, err
}

// isInteractive returns true if the program is running in the interactive
// terminal.
func isInteractive() bool {
	return term.IsTerminal(int(os.Stdout.Fd())) && term.IsTerminal(int(os.Stdin.Fd())) && os.Getenv("TERM") != "dumb"
}

func isRoot() bool {
	return os.Geteuid() == 0
}
