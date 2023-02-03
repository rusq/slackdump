package v1

import (
	"errors"
	"fmt"

	"github.com/AlecAivazis/survey/v2"

	"github.com/rusq/slackdump/v2/internal/app/config"
	"github.com/rusq/slackdump/v2/internal/ui"
	"github.com/rusq/slackdump/v2/internal/ui/ask"
)

var errExit = errors.New("exit")

var mainMenu = []struct {
	Name        string
	Description string
	Fn          func(p *params) error
}{
	{
		Name:        "Dump",
		Description: "save a list of conversations",
		Fn:          surveyDump,
	},
	{
		Name:        "Export",
		Description: "save the workspace or conversations in Slack Export format",
		Fn:          surveyExport,
	},
	{
		Name:        "List",
		Description: "list conversations or users on the screen",
		Fn:          surveyList,
	},
	{
		Name:        "Emojis",
		Description: "export all emojis from a workspace",
		Fn:          surveyEmojis,
	},
	{
		Name:        "Exit",
		Description: "Exit and return to the main menu.",
		Fn: func(*params) error {
			return errExit
		},
	},
}

func Interactive(p *params) error {
	var items = make([]string, len(mainMenu))
	for i := range mainMenu {
		items[i] = mainMenu[i].Name
	}

	mode := &survey.Select{
		Message: "What would you like to do?",
		Options: items,
		Description: func(value string, index int) string {
			return mainMenu[index].Description
		},
	}
	var resp string
	if err := survey.AskOne(mode, &resp); err != nil {
		return err
	}
	for _, mi := range mainMenu {
		if resp == mi.Name {
			return mi.Fn(p)
		}
	}
	// we should never get here.
	return errors.New("internal error: invalid choice")
}

func surveyList(p *params) error {
	qs := []*survey.Question{
		{
			Name:     "entity",
			Validate: survey.Required,
			Prompt: &survey.Select{
				Message: "List: ",
				Options: []string{"Conversations", "Users"},
				Description: func(value string, index int) string {
					return "List Slack " + value
				},
			},
		},
		{
			Name:     "format",
			Validate: survey.Required,
			Prompt: &survey.Select{
				Message: "Report format: ",
				Options: []string{config.OutputTypeText, config.OutputTypeJSON},
				Description: func(value string, index int) string {
					return "produce output in " + value + " format"
				},
			},
		},
	}

	mode := struct {
		Entity string
		Format string
	}{}

	var err error
	if err = survey.Ask(qs, &mode); err != nil {
		return err
	}

	switch mode.Entity {
	case "Conversations":
		p.appCfg.ListFlags.Channels = true
	case "Users":
		p.appCfg.ListFlags.Users = true
	}
	p.appCfg.Output.Format = mode.Format
	p.appCfg.Output.Filename, err = questOutputFile()
	return err
}

func surveyExport(p *params) error {
	var err error

	p.appCfg.ExportName, err = ui.StringRequire(
		"Output directory or ZIP file: ",
		"Enter the output directory or ZIP file name.  Add \".zip\" extension to save to a zip file.\nFor Mattermost, zip file is recommended.",
	)
	if err != nil {
		return err
	}
	p.appCfg.Input.List, err = ask.ConversationList("Conversations to export (leave empty or type ALL for full export): ")
	if err != nil {
		return err
	}
	p.appCfg.SlackConfig.DumpFiles, err = ui.Confirm("Export files?", true)
	if err != nil {
		return err
	}
	if p.appCfg.SlackConfig.DumpFiles {
		p.appCfg.ExportType, err = ask.ExportType()
		if err != nil {
			return err
		}
		p.appCfg.ExportToken, err = ui.String("Append export token (leave empty if none)", "export token will be appended to all file URLs.")
		if err != nil {
			return err
		}
	}

	return nil
}

func surveyDump(p *params) error {
	var err error
	p.appCfg.Input.List, err = ask.ConversationList("Enter conversations to dump: ")
	return err
}

// questOutputFile prints the output file question.
func questOutputFile() (string, error) {
	return ui.FileSelector(
		"Output file name (if empty - screen output): ",
		"Enter the filename to save the data to. Leave empty to print the results on the screen.",
		ui.WithDefaultFilename("-"),
	)
}

func surveyEmojis(p *params) error {
	p.appCfg.Emoji.Enabled = true
	var base string
	for {
		var err error
		base, err = ui.FileSelector("Enter directory or ZIP file name: ", "Emojis will be saved to this directory or ZIP file")
		if err != nil {
			return err
		}
		if base != "-" && base != "" {
			break
		}
		fmt.Println("invalid filename")
	}
	p.appCfg.Output.Base = base

	var err error
	p.appCfg.Emoji.FailOnError, err = ui.Confirm("Fail on download errors?", false)
	if err != nil {
		return err
	}
	return nil
}
