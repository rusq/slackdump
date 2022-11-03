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

	"github.com/rusq/dlog"

	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
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

Running this tool may be requested by developers.

<id> is the ID or URL of the workspace, for example "sdump" or 
https://sdump.slack.com.
`,
	CustomFlags: false,
	FlagMask:    cfg.OmitAll,
	PrintFlags:  true,
	RequireAuth: true,
	Commands:    nil,
}

type params struct {
	output string

	idOrURL string
}

var p params

func init() {
	CmdRawOutput.Run = runRawOutput
	CmdRawOutput.Flag.StringVar(&p.output, "o", "slackdump_raw.log", "output file")
}

func runRawOutput(ctx context.Context, cmd *base.Command, args []string) error {
	lg := dlog.FromContext(ctx)
	lg.SetPrefix("rawoutput ")

	if len(args) == 0 {
		CmdRawOutput.Flag.Usage()
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("missing ids or channel/thread links")
	}
	p.idOrURL = args[0]

	if err := run(ctx, p); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	return nil
}

const (
	domain  = "https://slack.com"
	baseURL = domain + "/api/"
)

func run(ctx context.Context, p params) error {
	prov, err := auth.FromContext(ctx)
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
