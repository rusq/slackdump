package ui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
)

func FileSelector(msg, descr string) (string, error) {
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
		overwrite, err := Confirm(fmt.Sprintf("File %q exists. Overwrite?", output), false)
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
