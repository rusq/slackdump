// Package nametmpl contains the name template logic.
package nametmpl

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/rusq/slackdump/v2/types"
	"github.com/slack-go/slack"
)

const filenameTmplName = "nametmpl"

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

// Compile checks the template for validness and compiles it returning the
// template and an error if any.
func Compile(t string) (*template.Template, error) {
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
