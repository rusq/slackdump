package diag

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/internal/app/appauth"
	"github.com/rusq/slackdump/v2/internal/chttp"
	"github.com/rusq/slackdump/v2/internal/structures"
)

var CmdRawOutput = &base.Command{
	Run:       nil, // populated by init to break the init cycle
	Wizard:    func(context.Context, *base.Command, []string) error { panic("not implemented") },
	UsageLine: "slackdump diag rawoutput [flags] <id>",
	Short:     "record raw API output",
	Long: `
Rawoutput produces a log file with the raw API output (as received from Slack
API).

Running this tool may be requested in response to a Github issue.
`,
	CustomFlags: true,
	FlagMask:    0,
	PrintFlags:  false,
	RequireAuth: false,
	Commands:    nil,
}

func init() {
	CmdRawOutput.Run = runRawOutput
}

type params struct {
	creds     appauth.SlackCreds
	output    string
	workspace string

	idOrURL string
}

func init() {
	CmdEzTest.Flag.Usage = func() {
		fmt.Fprintf(CmdEzTest.Flag.Output(), "usage: %s\n", CmdRawOutput.UsageLine)
		fmt.Fprintln(CmdEzTest.Flag.Output(), "Where `id' is an ID or URL of slack channel or thread.\n\nFlags:")
		CmdEzTest.Flag.PrintDefaults()
	}
}

func initFlags() params {
	var p params
	CmdEzTest.Flag.StringVar(&p.creds.Token, "token", os.Getenv("SLACK_TOKEN"), "slack token")
	CmdEzTest.Flag.StringVar(&p.creds.Cookie, "cookie", os.Getenv("COOKIE"), "slack cookie or path to a file with cookies")
	CmdEzTest.Flag.StringVar(&p.output, "o", "slackdump_raw.log", "output file")
	CmdEzTest.Flag.StringVar(&p.workspace, "w", "", "optional slack workspace name or URL")

	return p
}

func parseArgs(p *params, args []string) error {
	if err := CmdEzTest.Flag.Parse(args); err != nil {
		CmdEzTest.Flag.Usage()
		return err
	}
	if CmdEzTest.Flag.NArg() == 0 {
		CmdEzTest.Flag.Usage()
		return errors.New("missing ids or channel/thread links")
	}
	p.idOrURL = CmdEzTest.Flag.Arg(0)
	return nil
}

func runRawOutput(ctx context.Context, cmd *base.Command, args []string) {
	p := initFlags()
	if err := parseArgs(&p, args); err != nil {
		base.SetExitStatus(base.SInvalidParameters)
		fmt.Fprintln(os.Stderr, "Error: ", err)
		return
	}

	if err := run(ctx, p); err != nil {
		fmt.Fprintf(os.Stderr, "Error occurred: %s", err)
		return
	}
}

const (
	domain  = "https://slack.com"
	baseURL = domain + "/api/"
)

func run(ctx context.Context, p params) error {
	prov, err := appauth.InitProvider(ctx, cfg.CacheDir(), p.workspace, p.creds)
	if err != nil {
		return err
	}

	sl, err := structures.ParseLink(p.idOrURL)
	if err != nil {
		return err
	}
	cl := chttp.New(domain, chttp.ConvertCookies(prov.Cookies()), chttp.NewTransport(nil))
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
