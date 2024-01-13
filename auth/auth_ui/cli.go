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

// CLI is the archaic fallback UI for auth.
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
	workspace, err := prompt(w, "Enter Slack Workspace Name or URL: ", readln)
	if err != nil {
		return "", err
	}
	return Sanitize(workspace)
}

func (cl *CLI) RequestEmail(w io.Writer) (string, error) {
	return prompt(w, "Enter Email: ", readln)
}

func (cl *CLI) RequestPassword(w io.Writer, account string) (string, error) {
	defer fmt.Fprintln(w)
	return prompt(w, fmt.Sprintf("Enter Password for %s (won't be visible): ", account), readpwd)
}

func (cl *CLI) RequestLoginType(w io.Writer) (LoginType, error) {
	var types = []struct {
		name  string
		value LoginType
	}{
		{"Email", LHeadless},
		{"Google", LInteractive},
		{"Apple", LInteractive},
		{"Login with Single-Sign-On (SSO)", LInteractive},
		{"Other/Manual", LInteractive},
		{"Cancel", LCancel},
	}

	var idx int = -1
	for idx < 0 || idx >= len(types) {
		fmt.Fprintf(w, "Select login type:\n")
		for i, t := range types {
			fmt.Fprintf(w, "\t%d. %s\n", i+1, t.name)
		}
		fmt.Fprintf(w, "Enter number, and press Enter: ")

		_, err := fmt.Fscanf(os.Stdin, "%d", &idx)
		if err != nil {
			fmt.Fprintln(w, err)
			continue
		}

		idx -= 1 // adjusting for 0-index

		if idx < 0 || idx >= len(types) {
			fmt.Fprintln(w, "invalid login type")
		}
	}
	return types[idx].value, nil
}

func (*CLI) Stop() {}

func readln(r *os.File) (string, error) {
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func readpwd(f *os.File) (string, error) {
	pwd, err := term.ReadPassword(int(f.Fd()))
	if err != nil {
		return "", err
	}
	return string(pwd), nil
}

func prompt(w io.Writer, prompt string, readlnFn func(*os.File) (string, error)) (string, error) {
	for {
		fmt.Fprint(w, prompt)
		v, err := readlnFn(os.Stdin)
		if err != nil {
			return "", err
		}
		if v != "" {
			return v, nil
		}
		fmt.Fprintln(w, "input cannot be empty")
	}
}

func (*CLI) ConfirmationCode(email string) (code int, err error) {
	fmt.Printf("Enter confirmation code sent to %s: ", email)
	_, err = fmt.Fscanf(os.Stdin, "%d", &code)
	return
}
