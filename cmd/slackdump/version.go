package main

import (
	"context"
	"errors"
	"fmt"
	"runtime/trace"

	"golang.org/x/mod/semver"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/updater"
)

const sUnknown = "unknown"

var (
	version = sUnknown
	commit  = sUnknown
	date    = sUnknown
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

var ErrNoUpdates = errors.New("no updates")

func CheckUpdates(ctx context.Context) (*updater.Version, bool, error) {
	ctx, task := trace.NewTask(ctx, "CheckUpdates")
	defer task.End()

	if version == sUnknown {
		// development env
		return nil, false, nil
	}
	latest, err := updater.NewUpdater().Latest(ctx)
	if err != nil {
		return nil, false, err
	}
	return &latest, semver.Compare(version, latest.Version) > 0, nil
}
