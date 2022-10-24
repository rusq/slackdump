package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime/trace"
	"strings"

	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/help"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/list"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/man"
	v1 "github.com/rusq/slackdump/v2/cmd/slackdump/internal/v1"
	"github.com/rusq/tracer"
)

func init() {
	base.Slackdump.Commands = []*base.Command{
		v1.CmdV1,
		list.CmdList,

		man.ManLogin,
	}
}

func main() {
	flag.Usage = base.Usage
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		base.Usage()
	}

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
	if cmd.CustomFlags {
		args = args[1:]
	} else {
		cfg.SetBaseFlags(&cmd.Flag)
		cmd.Flag.Usage = func() { cmd.Usage() }
		cmd.Flag.Parse(args[1:])
		args = cmd.Flag.Args()
	}

	// maybe start trace
	if err := initTrace(cfg.TraceFile); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start trace: %s", err)
		base.SetExitStatus(1)
		return
	}

	ctx, task := trace.NewTask(context.Background(), "command")
	defer task.End()
	trace.Log(ctx, "command", fmt.Sprint("Running ", cmd.Name(), " command"))
	cmd.Run(ctx, cmd, args)
}

// initTrace initialises the tracing.  If the filename is not empty, the file
// will be opened, trace will write to that file.  Returns the stop function
// that must be called in the deferred call.  If the error is returned the stop
// function is nil.
func initTrace(filename string) error {
	if filename == "" {
		return nil
	}

	fmt.Fprintf(os.Stderr, "trace will be written to %q", filename)

	trc := tracer.New(filename)
	if err := trc.Start(); err != nil {
		return nil
	}

	stop := func() {
		if err := trc.End(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write the trace file: %s", err)
		}
	}
	base.AtExit(stop)
	return nil
}
