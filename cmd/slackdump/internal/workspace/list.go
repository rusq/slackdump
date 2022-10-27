package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rusq/dlog"

	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
)

var CmdListWsp = &base.Command{
	Run:       runList,
	Wizard:    nil,
	UsageLine: "slackdump workspace list [flags]",
	Short:     "list saved workspaces",
	Long: `
List allows to list Slack Workspaces, that you have previously authenticated in.
`,
	FlagMask:   flagmask,
	PrintFlags: true,
}

func runList(ctx context.Context, cmd *base.Command, args []string) {
	entries, err := filepath.Glob(filepath.Join(cfg.CacheDir(), "*.bin"))
	if err != nil {
		dlog.Printf("error trying to find existing workspaces: %s", err)
		base.SetExitStatus(base.SCacheError)
		return
	}
	if len(entries) == 0 {
		fmt.Println("No workspaces exist on this device.")
		fmt.Println("Run:  slackdump workspaces auth   to authenticate.")
		// TODO: do we want to ask user to authenticate?
		return
	}
	fmt.Printf("Workspaces in %q:\n\n", cfg.CacheDir())
	current, err := Current()
	if err != nil {
		dlog.Printf("error getting the current workspace: %s", err)
		base.SetExitStatus(base.SWorkspaceError)
	}
	for _, direntry := range entries {
		fmt.Println("\t" + formatWsp(direntry, current))
	}
	fmt.Println("\nCurrent workspace is marked with ' => '.")
}

func wspName(filename string) string {
	name := filepath.Base(filename)
	if name == defaultWspFilename {
		name = "default"
	} else {
		ext := filepath.Ext(name)
		name = name[:len(name)-len(ext)]
	}
	return name
}

func formatWsp(filename string, current string) string {
	timestamp := "unknown"
	if fi, err := os.Stat(filename); err == nil {
		timestamp = fi.ModTime().Format("2006-01-02 15:04:05")
	}
	name := wspName(filename)
	if filepath.Base(filename) == current {
		name = "=> " + name
	} else {
		name = "   " + name
	}

	return fmt.Sprintf("%s (last modified: %s)", name, timestamp)
}
