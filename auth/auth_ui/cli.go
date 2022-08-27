package auth_ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
)

type CLI struct{}

func (*CLI) instructions(w io.Writer) {
	const welcome = "Welcome to Slackdump EZ-Login 3000"
	underline := color.Set(color.Underline)
	fmt.Fprintf(w, "%s\n\n", underline.Sprint(welcome))
	fmt.Fprintf(w, "Please read these instructions carefully:\n\n")
	fmt.Fprintf(w, "1. Enter the slack workspace name or paste the URL of your slack workspace.\n\n   HINT: If https://example.slack.com is the Slack URL of your company,\n         then 'example' is the Slack Workspace name\n\n")
	fmt.Fprintf(w, "2. Browser will open, login as usual.\n\n")
	fmt.Fprintf(w, "3. Browser will close and slackdump will be authenticated.\n\n\n")
}

func (cl *CLI) RequestWorkspace(w io.Writer) (string, error) {
	cl.instructions(w)
	fmt.Fprint(w, "Enter Slack Workspace Name: ")
	workspace, err := readln(os.Stdin)
	if err != nil {
		return "", err
	}
	return workspace, nil
}

func (*CLI) Stop() {}

func readln(r io.Reader) (string, error) {
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
