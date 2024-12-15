package main

import (
	"context"
	"fmt"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
)

var (
	version = "unknown"
	commit  = "unknown"
	date    = "unknown"
)

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func init() {
	cfg.Version = cfg.BuildInfo{
		Version: version,
		Commit:  truncate(commit, 8), // to fit "Homebrew"
		Date:    date,
	}
}

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

func versionRun(context.Context, *base.Command, []string) error {
	fmt.Println(cfg.Version)
	return nil
}
