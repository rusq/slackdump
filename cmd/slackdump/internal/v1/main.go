package v1

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime/trace"
	"syscall"

	"github.com/fatih/color"
	"github.com/joho/godotenv"
	"github.com/rusq/dlog"
	"github.com/rusq/osenv/v2"
	"github.com/rusq/tracer"
	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/export"
	"github.com/rusq/slackdump/v2/internal/app"
	"github.com/rusq/slackdump/v2/internal/app/config"
	"github.com/rusq/slackdump/v2/internal/cache"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/logger"
)

const (
	envSlackToken     = "SLACK_TOKEN"
	envSlackCookie    = "COOKIE"
	envSlackFileToken = "SLACK_FILE_TOKEN"

	bannerFmt = "Slackdump %[1]s Copyright (c) 2018-%[2]s rusq (build: %s)\n\n"
)

var CmdV1 = &base.Command{
	Run:         runV1,
	Wizard:      nil, //initialised in init
	UsageLine:   "slackdump v1 [flags]",
	Short:       "slackdump legacy mode",
	CustomFlags: true,
	Long: `
Legacy mode
-----------

v1 command starts Slackdump in legacy mode, that supports all legacy flags.

Adjusting your scripts:
1. Replace all calls to "slackdump" with "slackdump -v".


For example, if you were exporting some workspace, and had this command:

  ./slackdump -export "my.zip" -format mattermost -download

Update it like this:

  ./slackdump v1 -export "my.zip" -format mattermost -download

To access legacy UI, run:

  ./slackdump v1

Compatibility with workspaces
-----------------------------

Legacy mode supports working on the default workspace.  Run:

  ./slackdump workspace select <workspace>

to set the default workspace.
`,
}

func init() {
	CmdV1.Wizard = func(ctx context.Context, cmd *base.Command, args []string) error {
		runV1(ctx, cmd, args)
		return nil
	}
}

// defFilenameTemplate is the default file naming template.
const defFilenameTemplate = "{{.ID}}{{ if .ThreadTS}}-{{.ThreadTS}}{{end}}"

var (
	build     = "dev"
	buildYear = "2077"
	commit    = "placeholder"
)

// secrets defines the names of the supported secret files that we load our
// secrets from.  Inexperienced Windows users might have bad experience trying
// to create .env file with the notepad as it will battle for having the
// "txt" extension.  Let it have it.
var secrets = []string{".env", ".env.txt", "secrets.txt"}

// params is the command line parameters
type params struct {
	appCfg    config.Params
	creds     cache.SlackCreds
	authReset bool

	traceFile string // trace file
	logFile   string //log file, if not specified, outputs to stderr.

	printVersion bool
	verbose      bool
}

func runV1(ctx context.Context, cmd *base.Command, args []string) error {
	banner(os.Stderr)
	loadSecrets(secrets)

	params, cfgErr := parseCmdLine(args[0:])

	if params.printVersion {
		fmt.Println(build)
		return nil
	}
	if params.authReset {
		if err := cache.AuthReset(params.appCfg.SlackConfig.CacheDir); err != nil {
			if !os.IsNotExist(err) {
				dlog.Printf("auth reset error: %s", err)
			}
		}
		if errors.Is(cfgErr, config.ErrNothingToDo) {
			// if no mode flag is specified - exit.
			dlog.Println("You have been logged out.")
			return nil
		}
	}

	if cfgErr == config.ErrNothingToDo {
		// if the user hasn't provided any required flags, let's offer
		// an interactive prompt to fill them.
		if err := Interactive(&params); err != nil {
			if err == errExit {
				return nil
			}
			base.SetExitStatus(base.SApplicationError)
			return err
		}
		if err := params.validate(); err != nil {
			base.SetExitStatus(base.SInvalidParameters)
			return err
		}
	} else if cfgErr != nil {
		return cfgErr
	}

	if err := run(context.Background(), params); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	return nil
}

// run runs the dumper.
func run(ctx context.Context, p params) error {
	// init logging and tracing
	lg, logStopFn, err := initLog(p.logFile, p.verbose)
	if err != nil {
		return err
	}
	defer logStopFn()
	ctx = dlog.NewContext(ctx, lg)

	// - setting the logger for the application.
	p.appCfg.Log = lg

	// - trace init
	if traceStopFn, err := initTrace(lg, p.traceFile); err != nil {
		return err
	} else {
		defer traceStopFn()
	}

	// initialise context with trace task.
	ctx, task := trace.NewTask(ctx, "main.run")
	defer task.End()

	provider, err := cache.InitProvider(ctx, p.appCfg.SlackConfig.CacheDir, "", p.creds)
	if err != nil {
		return err
	} else {
		p.creds = cache.SlackCreds{}
	}

	// trace startup parameters for debugging
	trace.Logf(ctx, "info", "params: input: %+v", p)

	// override default handler for SIGTERM and SIGQUIT signals.
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	// run the application
	if err := app.Run(ctx, p.appCfg, provider); err != nil {
		trace.Logf(ctx, "error", "app.Run: %s", err.Error())
		if isInvalidAuth(err) {
			return fmt.Errorf("failed to authenticate:  please double check that token/cookie values are correct (error: %w)", err)
		}
		return fmt.Errorf("application error: %w", err)
	}

	return nil
}

// initLog initialises the logging.  If the filename is not empty, the file will
// be opened, and the logger output will be switch to that file.  Returns the
// initialised logger, stop function and an error, if any.  The stop function
// must be called in the deferred call, it will close the log file, if it is
// open. If the error is returned the stop function is nil.
func initLog(filename string, verbose bool) (*dlog.Logger, func(), error) {
	lg := logger.Default
	lg.SetDebug(verbose)

	if filename == "" {
		return lg, func() {}, nil
	}

	lg.Debugf("log messages will be written to: %q", filename)
	lf, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return lg, nil, fmt.Errorf("failed to create the log file: %w", err)
	}
	lg.SetOutput(lf)

	stopFn := func() {
		if err := lf.Close(); err != nil {
			dlog.Printf("failed to close the log file: %s", err)
		}
	}
	return lg, stopFn, nil
}

// initTrace initialises the tracing.  If the filename is not empty, the file
// will be opened, trace will write to that file.  Returns the stop function
// that must be called in the deferred call.  If the error is returned the stop
// function is nil.
func initTrace(lg logger.Interface, filename string) (stop func(), err error) {
	if filename == "" {
		return func() {}, nil
	}

	lg.Printf("trace will be written to %q", filename)

	trc := tracer.New(filename)
	if err := trc.Start(); err != nil {
		return nil, err
	}
	return func() {
		if err := trc.End(); err != nil {
			lg.Printf("failed to write the trace file: %s", err)
		}
	}, nil
}

// isInvalidAuth returns true if err is Slack's invalid authentication error.
func isInvalidAuth(err error) bool {
	var ser slack.SlackErrorResponse
	return errors.As(err, &ser) && ser.Err == "invalid_auth"
}

// loadSecrets load secrets from the files in secrets slice.
func loadSecrets(files []string) {
	for _, f := range files {
		_ = godotenv.Load(f)
	}
}

func usage(fs *flag.FlagSet) func() {
	return func() {
		fmt.Fprintf(
			flag.CommandLine.Output(),
			"Slackdump saves conversations, threads and files from Slack.\n\n"+
				"This program comes with ABSOLUTELY NO WARRANTY;\n"+
				"This is free software, and you are welcome to redistribute it\n"+
				"under certain conditions.  Read LICENSE for more information.\n\n"+
				"Usage:  %s [flags] < -u | -c | [ID1 ID2 ... IDN] >\n"+
				"\twhere: ID is the conversation ID or URL Link to a conversation or thread\n"+
				"* NOTE: either `-u`, `-c` or URL or ID of the conversation must be specified\n\n"+
				"flags:\n",
			base.Executable())
		fs.PrintDefaults()
		fmt.Fprint(flag.CommandLine.Output(), color.HiYellowString(`
---------------------------------------------------------------
ATTENTION:  v1 command is deprecated in and will be removed
            in v3.1.0.
Run:
	`+base.Executable()+` help

to see the new command line interface help, or run without
parameters to use the Slackdump Wizard.
---------------------------------------------------------------
`),
		)
	}
}

// parseCmdLine parses the command line arguments.
func parseCmdLine(args []string) (params, error) {
	const zipHint = "\n(add .zip extension to save to a ZIP file)"

	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.Usage = usage(fs)

	var p = params{
		appCfg: config.Params{
			SlackConfig: slackdump.DefOptions,
			ExportType:  export.TNoDownload,
		},
	}

	// authentication
	fs.StringVar(&p.creds.Token, "t", osenv.Secret(envSlackToken, ""), "Specify slack `API_token`, (environment: "+envSlackToken+")")
	fs.StringVar(&p.creds.Cookie, "cookie", osenv.Secret(envSlackCookie, ""), "d= cookie `value` or a path to a cookie.txt file (environment: "+envSlackCookie+")")
	fs.BoolVar(&p.authReset, "auth-reset", false, "reset EZ-Login 3000 authentication.")

	// operation mode
	fs.BoolVar(&p.appCfg.ListFlags.Channels, "c", false, "same as -list-channels")
	fs.BoolVar(&p.appCfg.ListFlags.Channels, "list-channels", false, "list channels (aka conversations) and their IDs for export.")
	fs.BoolVar(&p.appCfg.ListFlags.Users, "u", false, "same as -list-users")
	fs.BoolVar(&p.appCfg.ListFlags.Users, "list-users", false, "list users and their IDs. ")
	// - export
	fs.StringVar(&p.appCfg.ExportName, "export", "", "`name` of the directory or zip file to export the Slack workspace to."+zipHint)
	fs.Var(&p.appCfg.ExportType, "export-type", "set the export type: 'standard' or 'mattermost' (default: standard)")
	fs.StringVar(&p.appCfg.ExportToken, "export-token", osenv.Secret(envSlackFileToken, ""), "Slack token that will be added to all file URLs, (environment: "+envSlackFileToken+")")
	// - emoji
	fs.BoolVar(&p.appCfg.Emoji.Enabled, "emoji", false, "dump all workspace emojis (set the base directory or zip file)")
	fs.BoolVar(&p.appCfg.Emoji.FailOnError, "emoji-fastfail", false, "fail on download error (if false, the download errors will be ignored\nand files will be skipped")

	// input-ouput options
	fs.StringVar(&p.appCfg.Output.Filename, "o", "-", "Output `filename` for users and channels.\nUse '-' for the Standard Output.")
	fs.StringVar(&p.appCfg.Output.Format, "r", "", "report `format`.  One of 'json' or 'text'")
	fs.StringVar(&p.appCfg.Output.Base, "base", "", "`name` of a directory or a file to save dumps to."+zipHint)
	fs.StringVar(&p.appCfg.FilenameTemplate, "ft", defFilenameTemplate, "output file naming template.")

	// options

	// - file download options
	fs.BoolVar(&p.appCfg.SlackConfig.DumpFiles, "f", slackdump.DefOptions.DumpFiles, "same as -download")
	fs.BoolVar(&p.appCfg.SlackConfig.DumpFiles, "download", slackdump.DefOptions.DumpFiles, "enable files download.")
	fs.IntVar(&p.appCfg.SlackConfig.Limits.Workers, "download-workers", slackdump.DefOptions.Limits.Workers, "number of file download worker threads.")
	fs.IntVar(&p.appCfg.SlackConfig.Limits.DownloadRetries, "dl-retries", slackdump.DefOptions.Limits.DownloadRetries, "rate limit retries for file downloads.")

	// - API request speed
	fs.IntVar(&p.appCfg.SlackConfig.Limits.Tier3.Retries, "t3-retries", slackdump.DefOptions.Limits.Tier3.Retries, "rate limit retries for conversation.")
	fs.UintVar(&p.appCfg.SlackConfig.Limits.Tier3.Boost, "t3-boost", slackdump.DefOptions.Limits.Tier3.Boost, "Tier-3 rate limiter boost in `events` per minute, will be added to the\nbase slack tier event per minute value.")
	fs.UintVar(&p.appCfg.SlackConfig.Limits.Tier3.Burst, "t3-burst", slackdump.DefOptions.Limits.Tier3.Burst, "Tier-3 rate limiter burst, allow up to `N` burst events per second.\nDefault value is safe.")
	fs.IntVar(&p.appCfg.SlackConfig.Limits.Tier2.Retries, "t2-retries", slackdump.DefOptions.Limits.Tier2.Retries, "rate limit retries for channel listing.")
	fs.UintVar(&p.appCfg.SlackConfig.Limits.Tier2.Boost, "t2-boost", slackdump.DefOptions.Limits.Tier2.Boost, "Tier-2 rate limiter boost in `events` per minute\n(affects users and channels).")
	fs.UintVar(&p.appCfg.SlackConfig.Limits.Tier2.Burst, "t2-burst", slackdump.DefOptions.Limits.Tier2.Burst, "Tier-2 rate limiter burst, allow up to `N` burst events per second.\n(affects users and channels).")

	fs.UintVar(&p.appCfg.SlackConfig.Limits.Tier3.Boost, "limiter-boost", slackdump.DefOptions.Limits.Tier3.Boost, "same as -t3-boost.")
	fs.UintVar(&p.appCfg.SlackConfig.Limits.Tier3.Burst, "limiter-burst", slackdump.DefOptions.Limits.Tier3.Burst, "same as -t3-burst.")

	// - API request size
	fs.IntVar(&p.appCfg.SlackConfig.Limits.Request.Conversations, "cpr", slackdump.DefOptions.Limits.Request.Conversations, "number of conversation `items` per request.")
	fs.IntVar(&p.appCfg.SlackConfig.Limits.Request.Channels, "npr", slackdump.DefOptions.Limits.Request.Channels, "number of `channels` per request.")
	fs.IntVar(&p.appCfg.SlackConfig.Limits.Request.Replies, "rpr", slackdump.DefOptions.Limits.Request.Replies, "number of `replies` per request.")

	// - cache controls
	fs.StringVar(&p.appCfg.SlackConfig.CacheDir, "cache-dir", cfg.CacheDir(), "slackdump cache directory")
	fs.StringVar(&p.appCfg.SlackConfig.UserCache.Filename, "user-cache-file", slackdump.DefOptions.UserCache.Filename, "user cache file`name`.")
	fs.DurationVar(&p.appCfg.SlackConfig.UserCache.MaxAge, "user-cache-age", slackdump.DefOptions.UserCache.MaxAge, "user cache lifetime `duration`. Set this to 0 to disable cache.")
	fs.BoolVar(&p.appCfg.SlackConfig.UserCache.Disabled, "no-user-cache", slackdump.DefOptions.UserCache.Disabled, "skip fetching users")

	// - time frame options
	fs.Var(&p.appCfg.Oldest, "dump-from", "`timestamp` of the oldest message to fetch from (i.e. 2020-12-31T23:59:59)")
	fs.Var(&p.appCfg.Latest, "dump-to", "`timestamp` of the latest message to fetch to (i.e. 2020-12-31T23:59:59)")

	// - main executable parameters
	fs.StringVar(&p.logFile, "log", osenv.Value("LOG_FILE", ""), "log `file`, if not specified, messages are printed to STDERR")
	fs.StringVar(&p.traceFile, "trace", osenv.Value("TRACE_FILE", ""), "trace `file` (optional)")
	fs.BoolVar(&p.printVersion, "V", false, "print version and exit")
	fs.BoolVar(&p.verbose, "v", osenv.Value("DEBUG", false), "verbose messages")

	os.Unsetenv(envSlackToken)
	os.Unsetenv(envSlackCookie)

	if err := fs.Parse(args); err != nil {
		return p, err
	}

	el, err := structures.NewEntityList(fs.Args())
	if err != nil {
		return p, err
	}

	p.appCfg.Input.List = el

	return p, p.validate()
}

// validate checks if the parameters are valid.
func (p *params) validate() error {
	if p.printVersion {
		return nil
	}
	return p.appCfg.Validate()
}

// banner prints the program banner.
func banner(w io.Writer) {
	fmt.Fprintf(w, bannerFmt, build, buildYear, trunc(commit, 7))
}

// trunc truncates string s to n chars
func trunc(s string, n uint) string {
	if uint(len(s)) <= n {
		return s
	}
	return s[:n]
}
