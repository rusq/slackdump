package diag

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/playwright-community/playwright-go"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/logger"
)

var CmdEzTest = &base.Command{
	Run:       runEzLoginTest,
	Wizard:    func(ctx context.Context, cmd *base.Command, args []string) error { panic("not implemented") },
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
}

type ezResult struct {
	Engine     string  `json:"engine,omitempty"`
	HasToken   bool    `json:"has_token,omitempty"`
	HasCookies bool    `json:"has_cookies,omitempty"`
	Err        *string `json:"error,omitempty"`
}

func init() {
	CmdEzTest.Flag.Usage = func() {
		fmt.Fprint(os.Stdout, "usage: slackdump tools eztest [flags]\n\nFlags:\n")
		CmdEzTest.Flag.PrintDefaults()
	}
}

func runEzLoginTest(ctx context.Context, cmd *base.Command, args []string) error {
	lg := logger.FromContext(ctx)

	wsp := cmd.Flag.String("w", "", "Slack `workspace` to login to.")
	legacy := cmd.Flag.Bool("legacy-browser", false, "run with playwright")

	if err := cmd.Flag.Parse(args); err != nil {
		base.SetExitStatus(base.SInvalidParameters)
		return err
	}

	if *wsp == "" {
		base.SetExitStatus(base.SInvalidParameters)
		cmd.Flag.Usage()
		return nil
	}

	var (
		res ezResult
	)

	if *legacy {
		res = tryPlaywrightAuth(ctx, *wsp)
	} else {
		res = tryRodAuth(ctx, *wsp)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(res); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	if res.Err == nil {
		lg.Println("OK")
	} else {
		lg.Println("ERROR")
		base.SetExitStatus(base.SApplicationError)
		return errors.New(*res.Err)
	}
	return nil

}

func tryPlaywrightAuth(ctx context.Context, wsp string) ezResult {
	var res = ezResult{Engine: "playwright"}

	if err := playwright.Install(&playwright.RunOptions{Browsers: []string{"firefox"}}); err != nil {
		res.Err = ptr(fmt.Sprintf("playwright installation error: %s", err))
		return res
	}

	prov, err := auth.NewBrowserAuth(ctx, auth.BrowserWithWorkspace(wsp))
	if err != nil {
		res.Err = ptr(err.Error())
		return res
	}

	res.HasToken = len(prov.SlackToken()) > 0
	res.HasCookies = len(prov.Cookies()) > 0
	if err != nil {
		res.Err = ptr(err.Error())
		return res
	}
	return res
}

func ptr[T any](t T) *T {
	return &t
}

func tryRodAuth(ctx context.Context, wsp string) ezResult {
	ret := ezResult{Engine: "rod"}
	prov, err := auth.NewRODAuth(ctx, auth.BrowserWithWorkspace(wsp))
	if err != nil {
		ret.Err = ptr(err.Error())
		return ret
	}
	ret.HasCookies = len(prov.Cookies()) > 0
	ret.HasToken = len(prov.SlackToken()) > 0
	return ret
}
