// Package base defines shared basic pieces of the slackdump command.
//
// The command subsystem is based on golang's `go` command implementation, which
// is BSD-licensed:
//
//	Copyright 2017 The Go Authors. All rights reserved.
//	Use of this source code is governed by a BSD-style
//	license that can be found in the LICENSE file.
package base

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
)

var CmdName string

// A Command is an implementation of a slackdump command.
type Command struct {
	// Run runs the command.
	// The args are the arguments after the command name.
	Run func(ctx context.Context, cmd *Command, args []string)

	// UsageLine is the one-line usage message.
	UsageLine string

	// Short is the short description shown in the 'go help' output.
	Short string

	// Long is the long message shown in the 'go help <this-command>' output.
	Long string

	// Flag is a set of flags specific to this command.
	Flag flag.FlagSet

	// CustomFlags indicates that the command will do its own
	// flag parsing.
	CustomFlags bool

	// PrintFlags indicates that generic help handler should print the
	// flags in the flagset.  Set it to false, if a Long lists all the flags.
	// It only matters for the commands that have no subcommands.
	PrintFlags bool

	// Commands lists the available commands and help topics.
	// The order here is the order in which they are printed by 'go help'.
	// Note that subcommands are in general best avoided.
	Commands []*Command
}

var Slackdump = &Command{
	UsageLine: "slackdump",
	Long:      `Slackdump is a tool for exporting Slack conversations, emojis, users, etc.`,
	// Commands initialised in main.
}

var exitStatus = 0
var exitMu sync.Mutex

func SetExitStatus(n int) {
	exitMu.Lock()
	if exitStatus < n {
		exitStatus = n
	}
	exitMu.Unlock()
}

var atExitFuncs []func()

func AtExit(f func()) {
	atExitFuncs = append(atExitFuncs, f)
}

func Exit() {
	for _, f := range atExitFuncs {
		f()
	}
	os.Exit(exitStatus)
}

// Runnable reports whether the command can be run; otherwise
// it is a documentation pseudo-command such as importpath.
func (c *Command) Runnable() bool {
	return c.Run != nil
}

// LongName returns the command's long name: all the words in the usage line between "go" and a flag or argument,
func (c *Command) LongName() string {
	name := c.UsageLine
	if i := strings.Index(name, " ["); i >= 0 {
		name = name[:i]
	}
	if name == "slackdump" {
		return ""
	}
	return strings.TrimPrefix(name, "slackdump ")
}

// Name returns the command's short name: the last word in the usage line before a flag or argument.
func (c *Command) Name() string {
	name := c.LongName()
	if i := strings.LastIndex(name, " "); i >= 0 {
		name = name[i+1:]
	}
	return name
}

// Usage is the usage-reporting function, filled in by package main
// but here for reference by other packages.
var Usage func()

func (c *Command) Usage() {
	fmt.Fprintf(os.Stderr, "usage: %s\n", c.UsageLine)
	fmt.Fprintf(os.Stderr, "Run 'slackdump help %s' for details.\n", c.LongName())
	SetExitStatus(2)
	Exit()
}
