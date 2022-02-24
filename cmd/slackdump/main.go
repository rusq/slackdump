package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/trace"

	"github.com/rusq/slackdump"
	"github.com/rusq/slackdump/internal/app"
	"github.com/rusq/slackdump/internal/tracer"

	"github.com/joho/godotenv"
	"github.com/rusq/dlog"
)

const (
	slackTokenEnv  = "SLACK_TOKEN"
	slackCookieEnv = "COOKIE"
)

const defFilenameTemplate = "{{.ID}}{{ if .ThreadTS}}-{{.ThreadTS}}{{end}}"

var build = "dev"

// secrets defines the names of the supported secret files that we load our
// secrets from.  Inexperienced windows users might have bad experience trying
// to create .env file with the notepad as it will battle for having the
// "txt" extension.  Let it have it.
var secrets = []string{".env", ".env.txt", "secrets.txt"}

// params is the command line parameters
type params struct {
	appCfg app.Config

	traceFile    string // trace file
	printVersion bool
	verbose      bool
}

func main() {
	loadSecrets(secrets)

	params, err := parseCmdLine(os.Args[1:])
	if err != nil {
		dlog.Fatal(err)
	}
	if params.printVersion {
		fmt.Println(build)
		return
	}
	if params.verbose {
		dlog.SetDebug(true)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx, params); err != nil {
		if params.verbose {
			dlog.Fatalf("%+v", err)
		} else {
			dlog.Fatal(err)
		}
	}
}

// run runs the dumper.
func run(ctx context.Context, p params) error {
	if p.traceFile != "" {
		dlog.Printf("enabling trace, will write to %q", p.traceFile)
		trc := tracer.New(p.traceFile)
		if err := trc.Start(); err != nil {
			return err
		}
		defer func() {
			if err := trc.End(); err != nil {
				dlog.Printf("failed to write the trace file: %s", err)
			}
		}()
	}

	ctx, task := trace.NewTask(ctx, "main.run")
	defer task.End()

	application, err := app.New(p.appCfg)
	if err != nil {
		trace.Logf(ctx, "error", "app.New: %s", err.Error())
		return err
	}

	// deleting creds to avoid logging them in the trace.
	p.appCfg.Creds = app.SlackCreds{}
	trace.Logf(ctx, "info", "params: input: %+v", p)

	if err := application.Run(ctx); err != nil {
		trace.Logf(ctx, "error", "app.Run: %s", err.Error())
		return err
	}
	return nil
}

// loadSecrets load secrets from the files in secrets slice.
func loadSecrets(files []string) {
	for _, f := range files {
		godotenv.Load(f)
	}
}

// parseCmdLine parses the command line arguments.
func parseCmdLine(args []string) (params, error) {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(
			flag.CommandLine.Output(),
			"Slackdump, %s"+
				"Slackdump dumps messages and files from slack using the provided api token.\n"+
				"Will create a number of files having the channel_id as a name.\n"+
				"Files are downloaded into a respective folder with channel_id name\n\n"+
				"Usage:  %s [flags] [channel_id1 ... channel_idN]\n\n",
			build, filepath.Base(os.Args[0]))
		fs.PrintDefaults()
	}

	var p params
	fs.StringVar(&p.appCfg.Creds.Token, "t", os.Getenv(slackTokenEnv), "Specify slack `API_token`, (environment: "+slackTokenEnv+")")
	fs.StringVar(&p.appCfg.Creds.Cookie, "cookie", os.Getenv(slackCookieEnv), "d= cookie `value` (environment: "+slackCookieEnv+")")

	fs.BoolVar(&p.appCfg.ListFlags.Channels, "c", false, "same as -list-channels")
	fs.BoolVar(&p.appCfg.ListFlags.Channels, "list-channels", false, "list channels (aka conversations) and their IDs for export.")
	fs.BoolVar(&p.appCfg.ListFlags.Users, "u", false, "same as -list-users")
	fs.BoolVar(&p.appCfg.ListFlags.Users, "list-users", false, "list users and their IDs. ")

	fs.StringVar(&p.appCfg.Input.Filename, "i", "", "specify the `input file` with Channel IDs or URLs to be used instead of giving the list on the command line, one per line.\nUse \"-\" to read input from STDIN.")
	fs.StringVar(&p.appCfg.Output.Filename, "o", "-", "Output `filename` for users and channels.  Use '-' for standard\nOutput.")
	fs.StringVar(&p.appCfg.Output.Format, "r", "", "report `format`.  One of 'json' or 'text'")

	fs.StringVar(&p.appCfg.FilenameTemplate, "ft", defFilenameTemplate, "output file naming template.")

	// options
	fs.BoolVar(&p.appCfg.Options.DumpFiles, "f", slackdump.DefOptions.DumpFiles, "same as -download")
	fs.BoolVar(&p.appCfg.Options.DumpFiles, "download", slackdump.DefOptions.DumpFiles, "enable files download.")
	fs.IntVar(&p.appCfg.Options.Workers, "download-workers", slackdump.DefOptions.Workers, "number of file download worker threads.")
	fs.IntVar(&p.appCfg.Options.Tier3Retries, "t3-retries", slackdump.DefOptions.Tier3Retries, "rate limit retries for conversation.")
	fs.IntVar(&p.appCfg.Options.Tier2Retries, "t2-retries", slackdump.DefOptions.Tier2Retries, "rate limit retries for channel listing.")
	fs.IntVar(&p.appCfg.Options.DownloadRetries, "dl-retries", slackdump.DefOptions.DownloadRetries, "rate limit retries for file downloads.")
	fs.IntVar(&p.appCfg.Options.ConversationsPerReq, "cpr", slackdump.DefOptions.ConversationsPerReq, "number of conversation `items` per request.")
	fs.UintVar(&p.appCfg.Options.Tier3Boost, "limiter-boost", slackdump.DefOptions.Tier3Boost, "same as -t3-boost.")
	fs.UintVar(&p.appCfg.Options.Tier3Boost, "t3-boost", slackdump.DefOptions.Tier3Boost, "Tier-3 rate limiter boost in `events` per minute, will be added to the\nbase slack tier event per minute value.")
	fs.UintVar(&p.appCfg.Options.Tier3Burst, "limiter-burst", slackdump.DefOptions.Tier3Burst, "same as -t3-burst.")
	fs.UintVar(&p.appCfg.Options.Tier3Burst, "t3-burst", slackdump.DefOptions.Tier3Burst, "allow up to `N` burst events per second.  Default value is safe.")
	fs.UintVar(&p.appCfg.Options.Tier2Boost, "t2-boost", slackdump.DefOptions.Tier2Boost, "Tier-2 limiter boost in `events` per minute (affects users and channels).")
	fs.UintVar(&p.appCfg.Options.Tier2Burst, "t2-burst", slackdump.DefOptions.Tier2Burst, "Tier-2 limiter burst in `events` (affects users and channels).")
	fs.StringVar(&p.appCfg.Options.UserCacheFilename, "user-cache-file", slackdump.DefOptions.UserCacheFilename, "user cache file`name`.")
	fs.DurationVar(&p.appCfg.Options.MaxUserCacheAge, "user-cache-age", slackdump.DefOptions.MaxUserCacheAge, "user cache lifetime `duration`. Set this to 0 to disable cache.")
	fs.BoolVar(&p.appCfg.Options.NoUserCache, "no-user-cache", slackdump.DefOptions.NoUserCache, "skip fetching users")

	fs.Var(&p.appCfg.Oldest, "dump-from", "`timestamp` of the oldest message to fetch from (i.e. 2020-12-31T23:59:59)")
	fs.Var(&p.appCfg.Latest, "dump-to", "`timestamp` of the latest message to fetch to (i.e. 2020-12-31T23:59:59)")

	// main parameters
	fs.StringVar(&p.traceFile, "trace", os.Getenv("TRACE_FILE"), "trace `file` (optional)")
	fs.BoolVar(&p.printVersion, "V", false, "print version and exit")
	fs.BoolVar(&p.verbose, "v", false, "verbose messages")

	os.Unsetenv(slackTokenEnv)
	os.Unsetenv(slackCookieEnv)

	if err := fs.Parse(args); err != nil {
		return p, err
	}

	p.appCfg.Input.List = fs.Args()

	return p, p.validate()
}

func (p *params) validate() error {
	if p.printVersion {
		return nil
	}
	return p.appCfg.Validate()
}
