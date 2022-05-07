package auth

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/playwright-community/playwright-go"

	"github.com/rusq/slackdump/v2/auth/browser"
)

var _ Provider = BrowserAuth{}

type BrowserAuth struct {
	simpleProvider
}

func NewBrowserAuth() (BrowserAuth, error) {
	if err := playwright.Install(&playwright.RunOptions{Browsers: []string{"chromium"}}); err != nil {
		return BrowserAuth{}, err
	}

	instructions(os.Stdout)
	workspace, err := requestWorkspace(os.Stdout)
	if err != nil {
		return BrowserAuth{}, err
	}

	auther, err := browser.New(workspace)
	if err != nil {
		return BrowserAuth{}, err
	}
	token, cookies, err := auther.Authenticate()
	if err != nil {
		return BrowserAuth{}, err
	}
	return BrowserAuth{
		simpleProvider: simpleProvider{
			token:   token,
			cookies: cookies,
		},
	}, nil
}

func instructions(w io.Writer) {
	const welcome = "Welcome to Slackdump EZ-Login 3000"
	underline := color.Set(color.Underline)
	fmt.Fprintf(w, "%s\n\n", underline.Sprint(welcome))
	fmt.Fprintf(w, "Please read these instructions carefully:\n\n")
	fmt.Fprintf(w, "1. Enter the slack workspace name or paste the URL of your slack workspace.\n\n   HINT: If https://example.slack.com is the Slack URL of your company,\n         then 'example' is the Slack Workspace name\n\n")
	fmt.Fprintf(w, "2. Browser will open, login as usual.\n\n")
	fmt.Fprintf(w, "3. Browser will close and slackdump will be authenticated.\n\n\n")
}

func requestWorkspace(w io.Writer) (string, error) {
	fmt.Fprint(w, "Enter Slack Workspace Name: ")
	workspace, err := readln(os.Stdin)
	if err != nil {
		return "", err
	}
	return sanitize(workspace)
}

func sanitize(workspace string) (string, error) {
	if !strings.Contains(workspace, ".slack.com") {
		return workspace, nil
	}
	if strings.HasPrefix(workspace, "https://") {
		uri, err := url.Parse(workspace)
		if err != nil {
			return "", err
		}
		workspace = uri.Host
	}
	// parse
	parts := strings.Split(workspace, ".")
	return parts[0], nil
}

func readln(r io.Reader) (string, error) {
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
