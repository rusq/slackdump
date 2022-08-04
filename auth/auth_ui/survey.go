package auth_ui

import (
	"fmt"
	"io"

	"github.com/AlecAivazis/survey/v2"
	"github.com/rusq/slackdump/v2/internal/app/ui"
)

type Survey struct{}

func (*Survey) RequestWorkspace(w io.Writer) (string, error) {
	workspace, err := ui.Input(
		"Enter Slack Workspace Name: ",
		"HELP:\n1. Enter the slack workspace name or paste the URL of your slack workspace.\n"+
			"2. Browser will open, login as usual.\n"+
			"3. Browser will close and slackdump will be authenticated.\n\n"+
			"This must be done only once.  The credentials are saved in an encrypted\nfile, and can be used only on this device.",
		survey.Required,
	)
	if err != nil {
		return "", err
	}
	fmt.Println("Please login in the browser...")
	return workspace, err
}

func (*Survey) Stop() {}
