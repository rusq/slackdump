package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"

	"github.com/rusq/slackdump/v2/export"
	"github.com/rusq/slackdump/v2/internal/app/config"
	"github.com/rusq/slackdump/v2/internal/app/ui"
	"github.com/rusq/slackdump/v2/internal/structures"
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

	p.appCfg.ExportType, err = questExportType()
	if err != nil {
		return err
	}

	p.appCfg.ExportName, err = ui.StringRequire(
		"Output directory or ZIP file: ",
		"Enter the output directory or ZIP file name.  Add \".zip\" extension to save to a zip file.\nFor Mattermost, zip file is recommended.",
	)
	if err != nil {
		return err
	}
	p.appCfg.Input.List, err = questConversationList("Conversations to export (leave empty for ALL): ")
	if err != nil {
		return err
	}
	p.appCfg.Options.DumpFiles, err = ui.Confirm("Export files?", true)
	if err != nil {
		return err
	}
	return nil
}

func questExportType() (export.ExportType, error) {
	mode := &survey.Select{
		Message: "Export type: ",
		Options: []string{export.TMattermost.String(), export.TStandard.String()},
		Description: func(value string, index int) string {
			descr := []string{
				"Mattermost bulk upload compatible export (see doc)",
				"Standard export format",
			}
			return descr[index]
		},
	}
	var resp string
	if err := survey.AskOne(mode, &resp); err != nil {
		return 0, err
	}
	var t export.ExportType
	t.Set(resp)
	return t, nil
}

func surveyDump(p *params) error {
	var err error
	p.appCfg.Input.List, err = questConversationList("Enter conversations to dump: ")
	return err
}

// questConversationList enquires the channel list.
func questConversationList(msg string) (*structures.EntityList, error) {
	for {
		chanStr, err := ui.String(
			msg,
			"Enter whitespace separated conversation IDs or URLs to export.\n"+
				"   - prefix with ^ (caret) to exclude the converation\n"+
				"   - prefix with @ to read the list of converations from the file.\n\n"+
				"For more details, see https://github.com/rusq/slackdump/blob/master/doc/usage-export.rst#providing-the-list-in-a-file",
		)
		if err != nil {
			return nil, err
		}
		if chanStr == "" {
			return new(structures.EntityList), nil
		}
		if el, err := structures.MakeEntityList(strings.Split(chanStr, " ")); err != nil {
			fmt.Println(err)
		} else {
			return el, nil
		}
	}
}

// questOutputFile prints the output file question.
func questOutputFile() (string, error) {
	return fileSelector(
		"Output file name (if empty - screen output): ",
		"Enter the filename to save the data to. Leave empty to print the results on the screen.",
	)
}

func fileSelector(msg, descr string) (string, error) {
	var q = &survey.Input{
		Message: msg,
		Suggest: func(partname string) []string {
			// thanks to AlecAivazis the for great example of this.
			files, _ := filepath.Glob(partname + "*")
			return files
		},
		Help: descr,
	}

	var (
		output string
	)
	for {
		if err := survey.AskOne(q, &output); err != nil {
			return "", err
		}
		if _, err := os.Stat(output); err != nil {
			break
		}
		overwrite, err := ui.Confirm(fmt.Sprintf("File %q exists. Overwrite?", output), false)
		if err != nil {
			return "", err
		}
		if overwrite {
			break
		}
	}
	if output == "" {
		output = "-"
	}
	return output, nil
}

func surveyEmojis(p *params) error {
	p.appCfg.Emoji = true
	var base string
	for {
		var err error
		base, err = fileSelector("Enter directory or ZIP file name: ", "Emojis will be saved to this directory or ZIP file")
		if err != nil {
			return err
		}
		if base != "-" && base != "" {
			break
		}
		fmt.Println("invalid filename")
	}
	p.appCfg.Output.Base = base
	return nil
}
