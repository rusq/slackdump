package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/rusq/slackdump/v2/edge"
	"github.com/rusq/slackdump/v2/internal/app"
)

type params struct {
	creds     app.SlackCreds
	output    string
	workspace string
}

var args params

func init() {
	flag.StringVar(&args.creds.Token, "token", os.Getenv("SLACK_TOKEN"), "slack token")
	flag.StringVar(&args.creds.Cookie, "cookie", os.Getenv("COOKIE"), "slack cookie or path to a file with cookies")
	flag.StringVar(&args.output, "o", "slackdump_raw.txt", "output file")
	flag.StringVar(&args.workspace, "w", "", "optional slack workspace name or URL")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags] <id>\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Where id is an id or URLs of slack channel or thread.\n\nFlags:")
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		fmt.Fprint(flag.CommandLine.Output(), "Error:  missing ids or channel/thread links\n")
		return
	}

	ctx := context.Background()
	if err := run(ctx, args); err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error occurred: %s", err)
		return
	}
	log.Println("ok")
}

func run(ctx context.Context, p params) error {
	prov, err := app.InitProvider(ctx, app.CacheDir(), p.workspace, p.creds)
	if err != nil {
		return err
	}

	cl := edge.HTTPClient(prov.SlackToken(), "https://slack.com", edge.ConvertCookies(prov.Cookies()))
	_ = cl

	return nil
}
