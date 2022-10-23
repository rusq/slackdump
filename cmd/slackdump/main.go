package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/trace"
	"strings"

	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/base"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/help"
	v1 "github.com/rusq/slackdump/v2/cmd/slackdump/internal/v1"
)

func init() {
	base.Slackdump.Commands = []*base.Command{
		v1.CmdV1,
	}
}

func main() {
	flag.Usage = base.Usage
	flag.Parse()
	log.SetFlags(0)

	args := flag.Args()
	base.CmdName = args[0]
	if args[0] == "help" {
		help.Help(os.Stdout, args[1:])
		return
	}
BigCmdLoop:
	for bigCmd := base.Slackdump; ; {
		for _, cmd := range bigCmd.Commands {
			if cmd.Name() != args[0] {
				continue
			}
			if len(cmd.Commands) > 0 {
				bigCmd = cmd
				args = args[1:]
				if len(args) == 0 {
					help.PrintUsage(os.Stderr, bigCmd)
					base.SetExitStatus(2)
					base.Exit()
				}
				if args[0] == "help" {
					// Accept 'go mod help' and 'go mod help foo' for 'go help mod' and 'go help mod foo'.
					help.Help(os.Stdout, append(strings.Split(base.CmdName, " "), args[1:]...))
					return
				}
				base.CmdName += " " + args[0]
				continue BigCmdLoop
			}
			if !cmd.Runnable() {
				continue
			}
			invoke(cmd, args)
			base.Exit()
			return
		}
		helpArg := ""
		if i := strings.LastIndex(base.CmdName, " "); i >= 0 {
			helpArg = " " + base.CmdName[:i]
		}
		fmt.Fprintf(os.Stderr, "slackdump %s: unknown command\nRun 'go help%s' for usage.\n", base.CmdName, helpArg)
		base.SetExitStatus(2)
		base.Exit()
	}
}

func init() {
	base.Usage = mainUsage
}

func mainUsage() {
	help.PrintUsage(os.Stderr, base.Slackdump)
	os.Exit(2)
}

func invoke(cmd *base.Command, args []string) {
	cmd.Flag.Usage = func() { cmd.Usage() }
	cmd.Flag.Parse(args[1:])
	args = cmd.Flag.Args()
	// maybe start trace
	ctx, task := trace.NewTask(context.Background(), fmt.Sprint("Running ", cmd.Name(), " command"))
	defer task.End()
	cmd.Run(ctx, cmd, args)
}
