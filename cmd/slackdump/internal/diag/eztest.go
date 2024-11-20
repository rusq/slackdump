package diag

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/playwright-community/playwright-go"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
)

var CmdEzTest = &base.Command{
	Run:       runEzLoginTest,
	UsageLine: "slack tools eztest",
	Short:     "EZ-Login 3000 test",
	Long: `
# EZ-Login 3000 Test tool
Eztest attempts to start EZ Login 3000 on the device.

The browser will open, and you will be offered to login to the workspace of your
choice.  On successful login it outputs the json with the test results.

You will see "OK" in the end if there were no issues, otherwise an error will
be printed and the test will be terminated.
`,
	CustomFlags: true,
	PrintFlags:  true,
}

type ezResult struct {
	Engine      string                  `json:"engine,omitempty"`
	HasToken    bool                    `json:"has_token,omitempty"`
	HasCookies  bool                    `json:"has_cookies,omitempty"`
	Err         *string                 `json:"error,omitempty"`
	Credentials *Credentials            `json:"credentials,omitempty"`
	Response    *slack.AuthTestResponse `json:"response,omitempty"`
}

type Credentials struct {
	Token   string         `json:"token,omitempty"`
	Cookies []*http.Cookie `json:"cookie,omitempty"`
}

type eztestOpts struct {
	printCreds bool
	wsp        string
	legacy     bool
}

var eztestFlags eztestOpts

func init() {
	CmdEzTest.Flag.Usage = func() {
		fmt.Fprint(os.Stdout, "usage: slackdump tools eztest [flags]\n\nFlags:\n")
		CmdEzTest.Flag.PrintDefaults()
	}
	CmdEzTest.Flag.BoolVar(&eztestFlags.printCreds, "p", false, "print credentials")
	CmdEzTest.Flag.BoolVar(&eztestFlags.legacy, "legacy-browser", false, "run with playwright")
	CmdEzTest.Flag.StringVar(&eztestFlags.wsp, "w", "", "Slack `workspace` to login to.")
}

func runEzLoginTest(ctx context.Context, cmd *base.Command, args []string) error {
	lg := cfg.Log

	if err := cmd.Flag.Parse(args); err != nil {
		base.SetExitStatus(base.SInvalidParameters)
		return err
	}

	if eztestFlags.wsp == "" {
		base.SetExitStatus(base.SInvalidParameters)
		cmd.Flag.Usage()
		return nil
	}

	var (
		res ezResult
	)

	if eztestFlags.legacy {
		res = tryPlaywrightAuth(ctx, eztestFlags.wsp, eztestFlags.printCreds)
	} else {
		res = tryRodAuth(ctx, eztestFlags.wsp, eztestFlags.printCreds)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(res); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	if res.Err == nil {
		lg.Info("OK")
	} else {
		lg.Error("ERROR", "error", *res.Err)
		base.SetExitStatus(base.SApplicationError)
		return errors.New(*res.Err)
	}
	return nil
}

func tryPlaywrightAuth(ctx context.Context, wsp string, populateCreds bool) ezResult {
	var ret = ezResult{Engine: "playwright"}

	if err := playwright.Install(&playwright.RunOptions{Browsers: []string{"firefox"}}); err != nil {
		ret.Err = ptr(fmt.Sprintf("playwright installation error: %s", err))
		return ret
	}

	prov, err := auth.NewBrowserAuth(ctx, auth.BrowserWithWorkspace(wsp))
	if err != nil {
		ret.Err = ptr(err.Error())
		return ret
	}

	ret.HasToken = len(prov.SlackToken()) > 0
	ret.HasCookies = len(prov.Cookies()) > 0
	if populateCreds {
		ret.Credentials = &Credentials{
			Token:   prov.SlackToken(),
			Cookies: prov.Cookies(),
		}
		resp, err := prov.Test(ctx)
		if err != nil {
			ret.Err = ptr(err.Error())
			return ret
		}
		ret.Response = resp
	}
	return ret
}

func ptr[T any](t T) *T { return &t }

func tryRodAuth(ctx context.Context, wsp string, populateCreds bool) ezResult {
	ret := ezResult{Engine: "rod"}
	prov, err := auth.NewRODAuth(ctx, auth.BrowserWithWorkspace(wsp))
	if err != nil {
		ret.Err = ptr(err.Error())
		return ret
	}

	ret.HasCookies = len(prov.Cookies()) > 0
	ret.HasToken = len(prov.SlackToken()) > 0
	if populateCreds {
		ret.Credentials = &Credentials{
			Token:   prov.SlackToken(),
			Cookies: prov.Cookies(),
		}
		resp, err := prov.Test(ctx)
		if err != nil {
			ret.Err = ptr(err.Error())
			return ret
		}
		ret.Response = resp
	}
	return ret
}
