package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/rusq/chttp"
	"github.com/rusq/slackdump/v2/auth/browser"
	"github.com/rusq/slackdump/v2/internal/app"
	"github.com/rusq/slackdump/v2/internal/structures"
)

type params struct {
	creds     app.SlackCreds
	output    string
	workspace string

	idOrURL string
}

var args params

func init() {
	flag.StringVar(&args.creds.Token, "token", os.Getenv("SLACK_TOKEN"), "slack token")
	flag.StringVar(&args.creds.Cookie, "cookie", os.Getenv("COOKIE"), "slack cookie or path to a file with cookies")
	flag.StringVar(&args.output, "o", "slackdump_raw.log", "output file")
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
	args.idOrURL = flag.Arg(0)

	ctx := context.Background()
	if err := run(ctx, args); err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Error occurred: %s", err)
		return
	}
}

const (
	domain  = "https://slack.com"
	baseURL = domain + "/api/"
)

func run(ctx context.Context, p params) error {
	prov, err := app.InitProvider(ctx, app.CacheDir(), p.workspace, p.creds, browser.Bfirefox, true)
	if err != nil {
		return err
	}

	sl, err := structures.ParseLink(args.idOrURL)
	if err != nil {
		return err
	}
	cl, err := chttp.New(domain, prov.Cookies())
	if err != nil {
		return err
	}
	if err := saveOutput(ctx, cl, p.output, prov.SlackToken(), sl); err != nil {
		return err
	}

	fmt.Println("OK")
	return nil
}

func saveOutput(ctx context.Context, cl *http.Client, filename string, token string, sl structures.SlackLink) error {
	w, err := maybeCreate(filename)
	if err != nil {
		return err
	}
	defer w.Close()

	log.SetOutput(w)
	log.SetPrefix(fmt.Sprintf("*** SLACKDUMP RAW [%s]: ", sl))

	if sl.IsThread() {
		return saveThread(ctx, cl, w, token, sl)
	} else {
		return saveConversation(ctx, cl, w, token, sl)
	}
}

func maybeCreate(filename string) (io.WriteCloser, error) {
	if filename == "" || filename == "-" {
		return os.Stdout, nil
	}

	f, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func saveThread(ctx context.Context, cl *http.Client, w io.Writer, token string, sl structures.SlackLink) error {
	v := url.Values{
		"token":   {token},
		"channel": {sl.Channel},
		"ts":      {sl.ThreadTS},
	}

	return rawDump(w, cl, "conversations.replies", v)
}

func saveConversation(ctx context.Context, cl *http.Client, w io.Writer, token string, sl structures.SlackLink) error {
	v := url.Values{
		"token":   {token},
		"channel": {sl.Channel},
	}

	return rawDump(w, cl, "conversations.history", v)
}

type apiResponse struct {
	Ok       bool `json:"ok,omitempty"`
	HasMore  bool `json:"has_more,omitempty"`
	Metadata struct {
		NextCursor string `json:"next_cursor,omitempty"`
	} `json:"response_metadata,omitempty"`
}

func rawDump(w io.Writer, cl *http.Client, ep string, v url.Values) error {
	var hasMore = true
	for i := 0; hasMore; i++ {
		log.Printf("request %5d start", i+1)

		var err error
		hasMore, err = sendReq(w, cl, ep, v)
		if err != nil {
			return err
		}

		fmt.Fprintln(w)
		log.Printf("request %5d end", i)
	}
	log.Println("dump completed")

	return nil
}

func sendReq(w io.Writer, cl *http.Client, ep string, v url.Values) (bool, error) {
	resp, err := cl.Get(baseURL + ep + "?" + v.Encode())
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	log.Print("request headers")
	resp.Header.Write(w)

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error while retrieving body: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		io.Copy(w, bytes.NewReader(data))
		return false, fmt.Errorf("server NOT OK: %s", resp.Status)
	}
	if len(data) == 0 {
		return false, nil
	}

	log.Print("request body")
	if _, err := io.Copy(w, bytes.NewReader(data)); err != nil {
		return false, err
	}

	// try decode to get the next cursor
	var r apiResponse
	if err := json.Unmarshal(data, &r); err != nil {
		log.Printf("not a json payload: %s", err)
		return false, nil
	}
	if r.HasMore {
		v.Set("cursor", r.Metadata.NextCursor)
		return true, nil
	}
	return false, nil
}
