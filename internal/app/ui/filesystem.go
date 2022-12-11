package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
)

type fileSelectorOpt struct {
	emptyFilename string // if set, the empty filename will be replaced to this value
}

func WithEmptyFilename(s string) Option {
	return func(so *inputOptions) {
		so.fileSelectorOpt.emptyFilename = s
	}
}

func FileSelector(msg, descr string, opt ...Option) (string, error) {
	var opts = defaultOpts().apply(opt...)

	var q = []*survey.Question{
		{
			Name: "filename",
			Prompt: &survey.Input{
				Message: msg,
				Suggest: func(partname string) []string {
					files, _ := filepath.Glob(partname + "*")
					return files
				},
				Help: descr,
			},
			Validate: func(ans interface{}) error {
				if ans.(string) != "" || opts.emptyFilename != "" {
					return nil
				}
				return errors.New("empty filename")
			},
		},
	}

	var resp struct {
		Filename string
	}
	for {
		if err := survey.Ask(q, &resp, opts.surveyOpts()...); err != nil {
			return "", err
		}
		if resp.Filename == "" && opts.emptyFilename != "" {
			resp.Filename = opts.emptyFilename
		}
		if _, err := os.Stat(resp.Filename); err != nil {
			break
		}
		overwrite, err := Confirm(fmt.Sprintf("File %q exists. Overwrite?", resp.Filename), false, opt...)
		if err != nil {
			return "", err
		}
		if overwrite {
			break
		}
	}
	return resp.Filename, nil
}
