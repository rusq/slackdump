package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/rusq/slackdump/v2/internal/app"
	"github.com/rusq/slackdump/v2/internal/structures"
)

var errExit = errors.New("exit")

func Interactive(p *params) error {
	mode := &survey.Select{
		Message: "Choose Slackdump Mode: ",
		Options: []string{"Dump", "Export", "List", "- Options", "Exit"},
	}
	var resp string
	if err := survey.AskOne(mode, &resp); err != nil {
		return err
	}
	var err error
	switch resp {
	case "Exit":
		err = errExit
	case "- Options":
		//
	case "Dump":
		//
		err = surveyDump(p)
	case "Export":
		//
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
				Options: []string{"Channels", "Users"},
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
					return "generate report in " + value + " format"
				},
			},
		},
	}

	mode := struct {
		Entity string
		Format string
	}{}

	if err := survey.Ask(qs, &mode); err != nil {
		return err
	}

	switch mode.Entity {
	case "Channels":
		p.appCfg.ListFlags.Channels = true
	case "Users":
		p.appCfg.ListFlags.Users = true
	}
	p.appCfg.Output.Format = mode.Format

	return nil
}

func surveyExport(p *params) error {
	var err error
	p.appCfg.ExportName, err = svMustInputString(
		"Output directory or ZIP file: ",
		"Enter the output directory or ZIP file name.  Add \".zip\" to save to zip file",
	)
	if err != nil {
		return err
	}
	p.appCfg.Input.List, err = surveyChanList()
	if err != nil {
		return err
	}
	return nil
}

func surveyDump(p *params) error {
	var err error
	p.appCfg.Input.List, err = surveyChanList()
	return err
}

func surveyInput(msg, help string, validator survey.Validator) (string, error) {
	qs := []*survey.Question{
		{
			Name:     "value",
			Validate: validator,
			Prompt: &survey.Input{
				Message: msg,
				Help:    help,
			},
		},
	}
	var m = struct {
		Value string
	}{}
	if err := survey.Ask(qs, &m); err != nil {
		return "", err
	}
	return m.Value, nil
}

func svMustInputString(msg, help string) (string, error) {
	return surveyInput(msg, help, survey.Required)
}

func svInputString(msg, help string) (string, error) {
	return surveyInput(msg, help, nil)
}

func surveyChanList() (*structures.EntityList, error) {
	for {
		chanStr, err := svInputString(
			"List of channels: ",
			"Enter whitespace separated channel IDs or URLs to export.\n"+
				"   - prefix with ^ (carret) to exclude the channel\n"+
				"   - prefix with @ to read the list of channels from the file.\n\n"+
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
