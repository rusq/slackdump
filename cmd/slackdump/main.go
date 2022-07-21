package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/trace"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/rusq/osenv/v2"
	"github.com/rusq/tracer"
	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/internal/app"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/logger"
)

const (
	slackTokenEnv  = "SLACK_TOKEN"
	slackCookieEnv = "COOKIE"

	bannerFmt = "Slackdump %[1]s Copyright (c) 2018-%[2]s rusq (build: %s)\n\n"
)

// defFilenameTemplate is the default file naming template.
const defFilenameTemplate = "{{.ID}}{{ if .ThreadTS}}-{{.ThreadTS}}{{end}}"

var (
	build     = "dev"
	buildYear = "2077"
	commit    = "placeholder"
)

// secrets defines the names of the supported secret files that we load our
// secrets from.  Inexperienced windows users might have bad experience trying
// to create .env file with the notepad as it will battle for having the
// "txt" extension.  Let it have it.
var secrets = []string{".env", ".env.txt", "secrets.txt"}

// params is the command line parameters
type params struct {
	appCfg app.Config
	creds  app.SlackCreds

	traceFile string // trace file
	logFile   string //log file, if not specified, outputs to stderr.

	printVersion bool
	verbose      bool
}

func main() {
	banner(os.Stderr)
	loadSecrets(secrets)

	params, err := parseCmdLine(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	if params.printVersion {
		fmt.Println(build)
		return
	}

	if err := run(context.Background(), params); err != nil {
		if params.verbose {
			log.Fatalf("%+v", err)
		} else {
			log.Fatal(err)
		}
	}
}

// run runs the dumper.
func run(ctx context.Context, p params) error {
	// init logging and tracing
	lg, logStopFn, err := initLog(p.logFile, p.verbose)
	if err != nil {
		return err
	}
	defer logStopFn()

	// - setting the logger for slackdump package
	p.appCfg.Options.Logger = lg

	// - trace init
	if traceStopFn, err := initTrace(lg, p.traceFile); err != nil {
		return err
	} else {
		defer traceStopFn()
	}

	// initialise context with trace task.
	ctx, task := trace.NewTask(ctx, "main.run")
	defer task.End()

	// init the authentication provider
	provider, err := p.creds.AuthProvider(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to initialise the auth provider: %w", err)
	} else {
		p.creds = app.SlackCreds{}
	}

	// trace startup parameters for debugging
	trace.Logf(ctx, "info", "params: input: %+v", p)

	// initialise the application
	application, err := app.New(p.appCfg, provider)
	if err != nil {
		trace.Logf(ctx, "error", "app.New: %s", err.Error())
		return fmt.Errorf("application failed to initialise: %w", err)
	}
	defer application.Close()

	// override default handler for SIGTERM and SIGQUIT signals.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// run the application
	if err := application.Run(ctx); err != nil {
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
func initLog(filename string, verbose bool) (logger.Interface, func(), error) {
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
			log.Printf("failed to close the log file: %s", err)
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

// parseCmdLine parses the command line arguments.
func parseCmdLine(args []string) (params, error) {
	const zipHint = "\n(add .zip extension to save to a ZIP file)"

	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.Usage = func() {
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
			filepath.Base(os.Args[0]))
		fs.PrintDefaults()
	}

	var p = params{
		appCfg: app.Config{
			Options: slackdump.DefOptions,
		},
	}

	// authentication
	fs.StringVar(&p.creds.Token, "t", osenv.Secret(slackTokenEnv, ""), "Specify slack `API_token`, (environment: "+slackTokenEnv+")")
	fs.StringVar(&p.creds.Cookie, "cookie", osenv.Secret(slackCookieEnv, ""), "d= cookie `value` or a path to a cookie.txt file (environment: "+slackCookieEnv+")")

	// operation mode
	fs.BoolVar(&p.appCfg.ListFlags.Channels, "c", false, "same as -list-channels")
	fs.BoolVar(&p.appCfg.ListFlags.Channels, "list-channels", false, "list channels (aka conversations) and their IDs for export.")
	fs.BoolVar(&p.appCfg.ListFlags.Users, "u", false, "same as -list-users")
	fs.BoolVar(&p.appCfg.ListFlags.Users, "list-users", false, "list users and their IDs. ")
	// - export
	fs.StringVar(&p.appCfg.ExportName, "export", "", "`name` of the directory or zip file to export the Slack workspace to."+zipHint)

	// input-ouput options
	fs.StringVar(&p.appCfg.Output.Filename, "o", "-", "Output `filename` for users and channels.\nUse '-' for the Standard Output.")
	fs.StringVar(&p.appCfg.Output.Format, "r", "", "report `format`.  One of 'json' or 'text'")
	fs.StringVar(&p.appCfg.Output.Base, "base", "", "`name` of a directory or a file to save dumps to."+zipHint)
	fs.StringVar(&p.appCfg.FilenameTemplate, "ft", defFilenameTemplate, "output file naming template.")

	// options

	// - file download options
	fs.BoolVar(&p.appCfg.Options.DumpFiles, "f", slackdump.DefOptions.DumpFiles, "same as -download")
	fs.BoolVar(&p.appCfg.Options.DumpFiles, "download", slackdump.DefOptions.DumpFiles, "enable files download.")
	fs.IntVar(&p.appCfg.Options.Workers, "download-workers", slackdump.DefOptions.Workers, "number of file download worker threads.")
	fs.IntVar(&p.appCfg.Options.DownloadRetries, "dl-retries", slackdump.DefOptions.DownloadRetries, "rate limit retries for file downloads.")

	// - API request speed
	fs.IntVar(&p.appCfg.Options.Tier3Retries, "t3-retries", slackdump.DefOptions.Tier3Retries, "rate limit retries for conversation.")
	fs.UintVar(&p.appCfg.Options.Tier3Boost, "t3-boost", slackdump.DefOptions.Tier3Boost, "Tier-3 rate limiter boost in `events` per minute, will be added to the\nbase slack tier event per minute value.")
	fs.UintVar(&p.appCfg.Options.Tier3Burst, "t3-burst", slackdump.DefOptions.Tier3Burst, "Tier-3 rate limiter burst, allow up to `N` burst events per second.\nDefault value is safe.")
	fs.IntVar(&p.appCfg.Options.Tier2Retries, "t2-retries", slackdump.DefOptions.Tier2Retries, "rate limit retries for channel listing.")
	fs.UintVar(&p.appCfg.Options.Tier2Boost, "t2-boost", slackdump.DefOptions.Tier2Boost, "Tier-2 rate limiter boost in `events` per minute\n(affects users and channels).")
	fs.UintVar(&p.appCfg.Options.Tier2Burst, "t2-burst", slackdump.DefOptions.Tier2Burst, "Tier-2 rate limiter burst, allow up to `N` burst events per second.\n(affects users and channels).")

	fs.UintVar(&p.appCfg.Options.Tier3Boost, "limiter-boost", slackdump.DefOptions.Tier3Boost, "same as -t3-boost.")
	fs.UintVar(&p.appCfg.Options.Tier3Burst, "limiter-burst", slackdump.DefOptions.Tier3Burst, "same as -t3-burst.")

	// - API request size
	fs.IntVar(&p.appCfg.Options.ConversationsPerReq, "cpr", slackdump.DefOptions.ConversationsPerReq, "number of conversation `items` per request.")
	fs.IntVar(&p.appCfg.Options.ChannelsPerReq, "npr", slackdump.DefOptions.ChannelsPerReq, "number of `channels` per request.")
	fs.IntVar(&p.appCfg.Options.RepliesPerReq, "rpr", slackdump.DefOptions.RepliesPerReq, "number of `replies` per request.")

	// - user cache controls
	fs.StringVar(&p.appCfg.Options.UserCacheFilename, "user-cache-file", slackdump.DefOptions.UserCacheFilename, "user cache file`name`.")
	fs.DurationVar(&p.appCfg.Options.MaxUserCacheAge, "user-cache-age", slackdump.DefOptions.MaxUserCacheAge, "user cache lifetime `duration`. Set this to 0 to disable cache.")
	fs.BoolVar(&p.appCfg.Options.NoUserCache, "no-user-cache", slackdump.DefOptions.NoUserCache, "skip fetching users")

	// - time frame options
	fs.Var(&p.appCfg.Oldest, "dump-from", "`timestamp` of the oldest message to fetch from (i.e. 2020-12-31T23:59:59)")
	fs.Var(&p.appCfg.Latest, "dump-to", "`timestamp` of the latest message to fetch to (i.e. 2020-12-31T23:59:59)")

	// - main executable parameters
	fs.StringVar(&p.logFile, "log", osenv.Value("LOG_FILE", ""), "log `file`, if not specified, messages are printed to STDERR")
	fs.StringVar(&p.traceFile, "trace", osenv.Value("TRACE_FILE", ""), "trace `file` (optional)")
	fs.BoolVar(&p.printVersion, "V", false, "print version and exit")
	fs.BoolVar(&p.verbose, "v", osenv.Value("DEBUG", false), "verbose messages")

	os.Unsetenv(slackTokenEnv)
	os.Unsetenv(slackCookieEnv)

	if err := fs.Parse(args); err != nil {
		return p, err
	}

	el, err := structures.MakeEntityList(fs.Args())
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
