package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/rusq/slackdump/v2/internal/app"
	"github.com/rusq/slackdump/v2/internal/app/ui"
	"github.com/rusq/slackdump/v2/internal/structures"
)

var errExit = errors.New("exit")

func Interactive(p *params) error {
	mode := &survey.Select{
		Message: "What would you like to do?",
		Options: []string{"Dump", "Export", "List", "Exit"},
		Description: func(value string, index int) string {
			descr := []string{
				"save a list of conversations",
				"save the workspace or conversations in Slack Export format",
				"list conversations or users on the screen",
				"exit Slackdump and return to OS",
			}
			return descr[index]
		},
	}
	var resp string
	if err := survey.AskOne(mode, &resp); err != nil {
		return err
	}
	var err error
	switch resp {
	case "Exit":
		err = errExit
	case "Dump":
		err = surveyDump(p)
	case "Export":
		err = surveyExport(p)
	case "List":
		err = surveyList(p)
	}
	return err
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
				Options: []string{app.OutputTypeText, app.OutputTypeJSON},
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
	p.appCfg.ExportName, err = ui.MustString(
		"Output directory or ZIP file: ",
		"Enter the output directory or ZIP file name.  Add \".zip\" to save to zip file",
	)
	if err != nil {
		return err
	}
	p.appCfg.Input.List, err = questConvoList()
	if err != nil {
		return err
	}
	return nil
}

func surveyDump(p *params) error {
	var err error
	p.appCfg.Input.List, err = questConvoList()
	return err
}

// questConvoList enquires the channel list.
func questConvoList() (*structures.EntityList, error) {
	for {
		chanStr, err := ui.String(
			"List conversations: ",
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
	var q = &survey.Input{
		Message: "Output file name (if empty - screen output): ",
		Suggest: func(partname string) []string {
			// thanks to AlecAivazis for great example of this.
			files, _ := filepath.Glob(partname + "*")
			return files
		},
		Help: "Enter the filename to save the data to. Leave empty to print the results on the screen.",
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
