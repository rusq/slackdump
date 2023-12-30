package auth_ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"golang.org/x/term"
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
	return Sanitize(workspace)
}

func (cl *CLI) RequestEmail(w io.Writer) (string, error) {
	fmt.Fprint(w, "Enter Email: ")
	username, err := readln(os.Stdin)
	if err != nil {
		return "", err
	}
	return username, nil
}

func (cl *CLI) RequestPassword(w io.Writer, account string) (string, error) {
	fmt.Fprintf(w, "Enter Password for %s (won't be visible): ", account)
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	fmt.Fprintln(w)
	return string(password), nil
}

func (cl *CLI) RequestLoginType(w io.Writer) (int, error) {
	var types = []struct {
		name  string
		value int
	}{
		{"Email", LoginEmail},
		{"Google", LoginSSO},
		{"Apple", LoginSSO},
		{"Login with Single-Sign-On (SSO)", LoginSSO},
		{"Other", LoginSSO},
	}

	var idx int
	for idx < 1 || idx > len(types)+1 {
		fmt.Fprintf(w, "Select login type:\n")
		for i, t := range types {
			fmt.Fprintf(w, "\t%d. %s\n", i+1, t.name)
		}
		fmt.Fprintf(w, "Enter number: ")
		_, err := fmt.Fscanf(os.Stdin, "%d", &idx)
		if err != nil {
			fmt.Fprintln(w, err)
			continue
		}
		if idx < 1 || idx > len(types)+1 {
			fmt.Fprintln(w, "invalid login type")
		}
	}
	return types[idx-1].value, nil
}

func (*CLI) Stop() {}

func readln(r io.Reader) (string, error) {
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
