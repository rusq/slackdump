package diag

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/playwright-community/playwright-go"

	"github.com/rusq/slackdump/v2/auth/browser"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/logger"
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
}

type result struct {
	HasToken   bool    `json:"has_token,omitempty"`
	HasCookies bool    `json:"has_cookies,omitempty"`
	Err        *string `json:"error,omitempty"`
}

func init() {
	CmdEzTest.Flag.Usage = func() {
		fmt.Fprint(os.Stdout, "usage: slackdump diag eztest [flags]\n\nFlags:\n")
		CmdEzTest.Flag.PrintDefaults()
	}
}

func runEzLoginTest(ctx context.Context, cmd *base.Command, args []string) error {
	lg := logger.FromContext(ctx)

	wsp := cmd.Flag.String("w", "", "Slack `workspace` to login to.")

	if err := cmd.Flag.Parse(args); err != nil {
		base.SetExitStatus(base.SInvalidParameters)
		return err
	}

	if *wsp == "" {
		base.SetExitStatus(base.SInvalidParameters)
		cmd.Flag.Usage()
		return nil
	}

	if err := playwright.Install(&playwright.RunOptions{Browsers: []string{"firefox"}}); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("playwright installation error: %w", err)
	}

	b, err := browser.New(*wsp)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}

	token, cookies, err := b.Authenticate(context.Background())
	r := result{
		HasToken:   len(token) > 0,
		HasCookies: len(cookies) > 0,
	}
	if err != nil {
		errStr := err.Error()
		r.Err = &errStr
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	if r.Err == nil {
		lg.Println("OK")
	} else {
		lg.Println("ERROR")
		base.SetExitStatus(base.SApplicationError)
		return errors.New(*r.Err)
	}
	return nil
}
