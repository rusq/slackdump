// This package is based on the Golang source code with some modifications.
//
// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package help

import (
	"fmt"
	"strings"
	"text/template"
)

var (
	helpTemplate = `{{if .Runnable}}usage: {{.UsageLine}}

{{end}}{{.Markdown | trim}}
`
	usageTemplate = `{{.Markdown | trim}}

Usage:

	{{.UsageLine}} <command> [arguments]

The commands are:
{{range .Commands}}{{if or (.Runnable) .Commands}}
	{{.Name | printf "%-11s"}} {{.Short}}{{end}}{{end}}

Use "slackdump help{{with .LongName}} {{.}}{{end}} <command>" for more information about a command.
{{if eq (.UsageLine) "slackdump"}}
Additional help topics:
{{range .Commands}}{{if and (not .Runnable) (not .Commands)}}
	{{.Name | printf "%-15s"}} {{.Short}}{{end}}{{end}}

Use "slackdump help{{with .LongName}} {{.}}{{end}} <topic>" for more information about that topic.
{{end}}
`
)

// RenderTemplate renders the given text using the variables
func RenderTemplate(tmpl string, data any) (string, error) {
	var b strings.Builder
	t, err := template.New("").Option("missingkey=error").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("parse: %w", err)
	}
	if err := t.Execute(&b, data); err != nil {
		return "", fmt.Errorf("execute: %w", err)
	}
	return b.String(), nil
}
