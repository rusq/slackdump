package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/trace"
	"text/tabwriter"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/internal/app/appauth"
	"github.com/rusq/slackdump/v2/logger"
)

var CmdWspList = &base.Command{
	UsageLine: "slackdump workspace list [flags]",
	Short:     "list saved workspaces",
	Long: base.Render(`
# Workspace List Command

**List** allows to list Slack Workspaces, that you have previously authenticated in.
`),
	FlagMask:   flagmask,
	PrintFlags: true,
}

const timeLayout = "2006-01-02 15:04:05"

var (
	bare = CmdWspList.Flag.Bool("b", false, "bare output format (just names)")
	all  = CmdWspList.Flag.Bool("a", false, "all information, including user")
)

func init() {
	CmdWspList.Run = runList
}

func runList(ctx context.Context, cmd *base.Command, args []string) error {
	m, err := appauth.NewManager(cfg.CacheDir())
	if err != nil {
		base.SetExitStatus(base.SCacheError)
		return err
	}

	formatter := printFull
	if *bare {
		formatter = printBare
	} else if *all {
		formatter = printAll
	}

	entries, err := m.List()
	if err != nil {
		if errors.Is(err, appauth.ErrNoWorkspaces) {
			base.SetExitStatus(base.SUserError)
			return errors.New("no authenticated workspaces, please run \"slackdump workspace new\"")
		}
		base.SetExitStatus(base.SCacheError)
		return err
	}
	current, err := m.Current()
	if err != nil {
		if !errors.Is(err, appauth.ErrNoDefault) {
			base.SetExitStatus(base.SWorkspaceError)
			return fmt.Errorf("error getting the current workspace: %s", err)
		}
		current = entries[0]
		if err := m.Select(current); err != nil {
			base.SetExitStatus(base.SWorkspaceError)
			return fmt.Errorf("error setting the current workspace: %s", err)
		}

	}

	formatter(m, current, entries)
	return nil
}

const defMark = "=>"

func printAll(m *appauth.Manager, current string, wsps []string) {
	ctx, task := trace.NewTask(context.Background(), "printAll")
	defer task.End()

	tw := tabwriter.NewWriter(os.Stdout, 2, 8, 1, ' ', 0)
	defer tw.Flush()

	fmt.Fprintln(tw,
		"C\tname\tfilename\tmodified\tteam\tuser\terror\n"+
			"-\t-------\t------------\t-------------------\t---------\t--------\t-----")
	cfg.SlackOptions.Logger = logger.Silent
	cfg.SlackOptions.NoUserCache = true
	for _, name := range wsps {
		curr := ""
		if current == name {
			curr = "*"
		}
		fi, err := m.FileInfo(name)
		if err != nil {
			fmt.Fprintf(tw, "%s\t%s\t\t\t\t\t%s\n", curr, name, err)
			continue
		}
		info, err := userInfo(ctx, m, name)
		if err != nil {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t\t\t%s\n", curr, name, fi.Name(), fi.ModTime().Format(timeLayout), err)
			continue
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", curr, name, fi.Name(), fi.ModTime().Format(timeLayout), info.Team, info.User, "OK")
	}
}

func userInfo(ctx context.Context, m *appauth.Manager, name string) (*slack.AuthTestResponse, error) {
	prov, err := m.Auth(ctx, name, appauth.SlackCreds{})
	if err != nil {
		return nil, err
	}
	sess, err := slackdump.NewWithOptions(ctx, prov, cfg.SlackOptions)
	if err != nil {
		return nil, err
	}
	return sess.Client().AuthTest()
}

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
		timestamp = fi.ModTime().Format(timeLayout)
		filename = fi.Name()
	}
	if name == current {
		name = defMark + " " + name
	} else {
		name = "   " + name
	}

	return fmt.Sprintf("%s (file: %s, last modified: %s)", name, filename, timestamp)
}

func printBare(_ *appauth.Manager, current string, workspaces []string) {
	for _, name := range workspaces {
		if current == name {
			fmt.Print("*")
		}
		fmt.Println(name)
	}
}
