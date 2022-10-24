// This package is based on the Golang source code with some modifications.
//
// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package help

var (
	helpTemplate = `{{if .Runnable}}usage: {{.UsageLine}}

{{end}}{{.Long | trim}}
`
	usageTemplate = `{{.Long | trim}}

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
	documentationTemplate = `{{range .}}{{if .Short}}{{.Short | capitalize}}

{{end}}{{if .Commands}}` + usageTemplate + `{{else}}{{if .Runnable}}Usage:

	{{.UsageLine}}

{{end}}{{.Long | trim}}


{{end}}{{end}}`
)
