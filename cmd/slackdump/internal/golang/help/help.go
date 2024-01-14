// This package is based on the Golang source code with some modifications.
//
// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package help implements "slackdump help" command.
package help

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
)

func PrintUsage(w io.Writer, cmd *base.Command) {
	bw := bufio.NewWriter(w)
	tmpl(bw, usageTemplate, cmd)
	bw.Flush()
}

// tmpl executes the given template text on data, writing the result to w.
func tmpl(w io.Writer, text string, data any) {
	t := template.New("top")
	t.Funcs(template.FuncMap{"trim": strings.TrimSpace, "capitalize": capitalize})
	template.Must(t.Parse(text))
	ew := &errWriter{w: w}
	err := t.Execute(ew, data)
	if ew.err != nil {
		// I/O error writing. Ignore write on closed pipe.
		if strings.Contains(ew.err.Error(), "pipe") {
			base.SetExitStatus(1)
			base.Exit()
		}
		log.Fatalf("writing output: %v", ew.err)
	}
	if err != nil {
		panic(err)
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToTitle(r)) + s[n:]
}

// An errWriter wraps a writer, recording whether a write error occurred.
type errWriter struct {
	w   io.Writer
	err error
}

func (w *errWriter) Write(b []byte) (int, error) {
	n, err := w.w.Write(b)
	if err != nil {
		w.err = err
	}
	return n, err
}

// Help implements the 'help' command.
func Help(w io.Writer, args []string) {
	cmd := base.Slackdump
Args:
	for i, arg := range args {
		for _, sub := range cmd.Commands {
			if sub.Name() == arg {
				cmd = sub
				continue Args
			}
		}

		// helpSuccess is the help command using as many args as possible that would succeed.
		helpSuccess := "slackdump help"
		if i > 0 {
			helpSuccess += " " + strings.Join(args[:i], " ")
		}
		fmt.Fprintf(os.Stderr, "slackdump help %s: unknown help topic. Run '%s'.\n", strings.Join(args, " "), helpSuccess)
		base.SetExitStatus(2) // failed at 'slackdump help cmd'
		base.Exit()
	}

	if len(cmd.Commands) > 0 {
		PrintUsage(os.Stdout, cmd)
	} else {
		tmpl(os.Stdout, helpTemplate, cmd)
		if cmd.PrintFlags {
			fmt.Fprintln(os.Stdout, "\nFlags:")
			if !cmd.CustomFlags {
				cfg.SetBaseFlags(&cmd.Flag, cmd.FlagMask)
			}
			cmd.Flag.PrintDefaults()
		}
	}
	// not exit 2: succeeded at 'slackdump help cmd'.
}
