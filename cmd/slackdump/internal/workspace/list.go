package workspace

import (
	"context"
	"fmt"

	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/internal/app/appauth"
)

var CmdWspList = &base.Command{
	UsageLine: "slackdump workspace list [flags]",
	Short:     "list saved workspaces",
	Long: `
List allows to list Slack Workspaces, that you have previously authenticated in.
`,
	FlagMask:   flagmask,
	PrintFlags: true,
}

var (
	bare = CmdWspList.Flag.Bool("b", false, "bare output format (just names)")
)

func init() {
	CmdWspList.Run = runList
}

func runList(ctx context.Context, cmd *base.Command, args []string) {
	m, err := appauth.NewManager(cfg.CacheDir())
	if err != nil {
		base.SetExitStatusMsg(base.SCacheError, err)
		return
	}

	formatter := printFull
	if *bare {
		formatter = printBare
	}

	entries, err := m.List()
	if err != nil {
		base.SetExitStatusMsg(base.SCacheError, err)
		return
	}
	current, err := m.Current()
	if err != nil {
		base.SetExitStatusMsg(base.SWorkspaceError, fmt.Sprintf("error getting the current workspace: %s", err))
		return
	}

	formatter(m, current, entries)
}

const defMark = "=>"

func printFull(m *appauth.Manager, current string, wsps []string) {
	fmt.Printf("Workspaces in %q:\n\n", cfg.CacheDir())

	for _, name := range wsps {
		fmt.Println("\t" + formatWsp(m, current, name))
	}
	fmt.Printf("\nCurrent workspace is marked with ' %s '.\n", defMark)
}

func formatWsp(m *appauth.Manager, current string, name string) string {
	timestamp := "unknown"
	filename := "-"
	if fi, err := m.FileInfo(name); err == nil {
		timestamp = fi.ModTime().Format("2006-01-02 15:04:05")
		filename = fi.Name()
	}
	if name == current {
		name = defMark + " " + name
	} else {
		name = "   " + name
	}

	return fmt.Sprintf("%s (file: %s, last modified: %s)", name, filename, timestamp)
}

func printBare(_ *appauth.Manager, _ string, workspaces []string) {
	for _, name := range workspaces {
		fmt.Println(name)
	}
}
