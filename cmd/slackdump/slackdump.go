package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/rusq/slackdump/internal/app"
	"github.com/rusq/slackdump/internal/tracer"

	"github.com/joho/godotenv"
	"github.com/rusq/dlog"
)

const (
	slackTokenEnv  = "SLACK_TOKEN"
	slackCookieEnv = "COOKIE"
)

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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx, params); err != nil {
		dlog.Fatal(err)
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

	app, err := app.New(p.appCfg)
	if err != nil {
		return err
	}

	return app.Run(ctx)
}

// loadSecrets load secrets from the files in secrets slice.
func loadSecrets(files []string) {
	for _, f := range files {
		godotenv.Load(f)
	}
}

// parseCmdLine parses the command line arguments.
func parseCmdLine(args []string) (params, error) {
	fs := flag.NewFlagSet("", flag.ExitOnError)
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
	fs.BoolVar(&p.appCfg.ListFlags.Channels, "c", false, "ListFlags channels (aka conversations) and their IDs for export.")
	fs.BoolVar(&p.appCfg.ListFlags.Users, "u", false, "ListFlags users and their IDs. ")
	fs.BoolVar(&p.appCfg.IncludeFiles, "f", false, "enable files download")
	fs.IntVar(&p.appCfg.Boost, "limiter-boost", 120, "rate limiter boost in `events` per minute, will be added to the\nbase slack tier event per minute value.")
	fs.StringVar(&p.appCfg.Output.Filename, "o", "-", "Output `filename` for users and channels.  Use '-' for standard\nOutput.")
	fs.StringVar(&p.appCfg.Output.Format, "r", "", "report `format`.  One of 'json' or 'text'")
	fs.StringVar(&p.appCfg.Creds.Token, "t", os.Getenv(slackTokenEnv), "Specify slack `API_token`, (environment: "+slackTokenEnv+")")
	fs.StringVar(&p.appCfg.Creds.Cookie, "cookie", os.Getenv(slackCookieEnv), "d= cookie `value` (environment: "+slackCookieEnv+")")
	fs.StringVar(&p.traceFile, "trace", os.Getenv("TRACE_FILE"), "trace `file` (optional)")
	fs.BoolVar(&p.printVersion, "V", false, "print version and exit")

	os.Unsetenv(slackTokenEnv)
	os.Unsetenv(slackCookieEnv)

	if err := fs.Parse(args); err != nil {
		return p, err
	}

	p.appCfg.ChannelIDs = fs.Args()

	return p, p.validate()
}

func (p *params) validate() error {
	if p.printVersion {
		return nil
	}
	return p.appCfg.Validate()
}
