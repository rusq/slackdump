package main

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"

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
		Description: "exit Slackdump and return to the OS",
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

	var options []huh.Option[string]
	for _, mi := range mainMenu {
		options = append(options, huh.NewOption(fmt.Sprintf("%-10s - %s", mi.Name, mi.Description), mi.Name))
	}

	var resp string
	q := huh.NewSelect[string]().Options(options...).Title("What would you like to do?").Value(&resp)
	if err := q.Run(); err != nil {
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
	mode := struct {
		Entity string
		Format string
	}{}
	form := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("Choose what to list").
			Value(&mode.Entity).
			Options(
				huh.NewOption("Conversations", "Conversations"),
				huh.NewOption("Users", "Users"),
			),
		huh.NewSelect[string]().
			Title("Choose the output format").
			Value(&mode.Format).
			Options(
				huh.NewOption(config.OutputTypeJSON, config.OutputTypeJSON),
				huh.NewOption(config.OutputTypeText, config.OutputTypeText),
			),
	))

	if err := form.Run(); err != nil {
		return err
	}

	switch mode.Entity {
	case "Conversations":
		p.appCfg.ListFlags.Channels = true
	case "Users":
		p.appCfg.ListFlags.Users = true
	}
	p.appCfg.Output.Format = mode.Format
	var err error
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
	p.appCfg.Options.DumpFiles, err = ui.Confirm("Export files?", true)
	if err != nil {
		return err
	}
	if p.appCfg.Options.DumpFiles {
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
