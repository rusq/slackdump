package main

import (
	"context"
	"fmt"

	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
)

var CmdVersion = &base.Command{
	UsageLine: "version",
	Short:     "print version and exit",
	Long: `
# Version Command

Prints version and exits, not much else to say.

And by the way, version is: ` + version + `, commit: ` + commit + `, built on ` + date + `.
`,
	Run: versionRun,
}

func versionRun(ctx context.Context, cmd *base.Command, args []string) error {
	printVersion()
	return nil
}

func printVersion() {
	fmt.Printf("Slackdump %s (commit: %s) built on: %s\n", version, commit, date)
}
