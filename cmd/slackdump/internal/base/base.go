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
)

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
}
