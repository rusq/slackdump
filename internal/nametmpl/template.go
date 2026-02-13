// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// Package nametmpl contains the name template logic.
package nametmpl

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/types"
)

const filenameTmplName = "nametmpl"

// Default is the default file naming template.
const Default = `{{.ID}}{{ if .ThreadTS}}-{{.ThreadTS}}{{end}}.json`

// let's define some markers
const (
	mNotOK     = "$$ERROR$$"   // not allowed at all
	mOK        = "$$OK$$"      // required
	mPartialOK = "$$PARTIAL$$" // partial (only goes well with OK)
)

// marking all the fields we want with OK, all the rest (the ones we DO NOT
// WANT) with NotOK.
var tc = types.Conversation{
	Name:     mOK,
	ID:       mOK,
	Messages: []types.Message{{Message: slack.Message{Msg: slack.Msg{Channel: mNotOK}}}},
	ThreadTS: mPartialOK,
}

type Template struct {
	t *template.Template
}

// New returns the template from a string.
func New(t string) (*Template, error) {
	tmpl, err := compile(t)
	if err != nil {
		return nil, err
	}
	return &Template{tmpl}, nil
}

// NewDefault returns the default template.
func NewDefault() *Template {
	t, err := New(Default)
	if err != nil {
		panic(err)
	}
	return t
}

// Compile checks the template for validness and compiles it returning the
// template and an error if any.
func compile(t string) (*template.Template, error) {
	tmpl, err := template.New(filenameTmplName).Parse(t)
	if err != nil {
		return nil, err
	}
	// are you ready for some filth? Here we go!

	// now we render the template and check for OK/NotOK values in the output.
	var buf strings.Builder
	if err := tmpl.ExecuteTemplate(&buf, filenameTmplName, tc); err != nil {
		return nil, err
	}
	if strings.Contains(buf.String(), mNotOK) || len(buf.String()) == 0 {
		return nil, fmt.Errorf("invalid fields in the template: %q", t)
	}
	if !strings.Contains(buf.String(), mOK) {
		// must contain at least one OK
		return nil, fmt.Errorf("this does not resolve to anything useful: %q", t)
	}
	return tmpl, nil
}

// Execute executes the template and returns the result.  It panics if the
// template cannot be executed, but please note that the template is checked
// for validity at compile time.
func (t *Template) Execute(c *types.Conversation) string {
	var buf strings.Builder
	if err := t.t.ExecuteTemplate(&buf, filenameTmplName, c); err != nil {
		panic(err)
	}
	return buf.String()
}

func Must(s string, err error) string {
	if err != nil {
		panic(err)
	}
	return s
}
