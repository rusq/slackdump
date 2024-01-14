package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/trace"
	"strings"
	"text/tabwriter"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/cache"
	"github.com/rusq/slackdump/v3/logger"
)

var CmdWspList = &base.Command{
	UsageLine: baseCommand + " list [flags]",
	Short:     "list saved authentication information",
	Long: `
# Auth List Command

**List** allows to list Slack Workspaces, that you have previously authenticated in.
`,
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
	m, err := cache.NewManager(cfg.CacheDir())
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
		if errors.Is(err, cache.ErrNoWorkspaces) {
			base.SetExitStatus(base.SUserError)
			return errors.New("no authenticated workspaces, please run \"slackdump " + baseCommand + " new\"")
		}
		base.SetExitStatus(base.SCacheError)
		return err
	}
	current, err := m.Current()
	if err != nil {
		if !errors.Is(err, cache.ErrNoDefault) {
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

func printAll(m manager, current string, wsps []string) {
	ctx, task := trace.NewTask(context.Background(), "printAll")
	defer task.End()

	tw := tabwriter.NewWriter(os.Stdout, 2, 8, 1, ' ', 0)
	defer tw.Flush()

	var hdrItems = []hdrItem{
		{"C", 1},
		{"name", 7},
		{"filename", 12},
		{"modified", 19},
		{"team", 9},
		{"user", 8},
		{"error", 5},
	}

	fmt.Fprintln(tw, printHeader(hdrItems...))

	// TODO: Concurrent pipeline.
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

type hdrItem struct {
	name string
	size int
}

func (h *hdrItem) String() string {
	return h.name
}

func (h *hdrItem) Size() int {
	if h.size == 0 {
		h.size = len(h.String())
	}
	return h.size
}

func (h *hdrItem) Underline(char ...string) string {
	if len(char) == 0 {
		char = []string{"-"}
	}
	return strings.Repeat(char[0], h.Size())
}

func printHeader(hi ...hdrItem) string {
	var sb strings.Builder
	for i, h := range hi {
		if i > 0 {
			sb.WriteByte('\t')
		}
		sb.WriteString(h.String())
	}
	sb.WriteByte('\n')
	for i, h := range hi {
		if i > 0 {
			sb.WriteByte('\t')
		}
		sb.WriteString(h.Underline())
	}
	return sb.String()
}

func userInfo(ctx context.Context, m manager, name string) (*slack.AuthTestResponse, error) {
	prov, err := m.Auth(ctx, name, cache.SlackCreds{})
	if err != nil {
		return nil, err
	}
	sess, err := slackdump.New(ctx, prov, slackdump.WithLogger(logger.Silent))
	if err != nil {
		return nil, err
	}
	return sess.Client().AuthTest()
}

func printFull(m manager, current string, wsps []string) {
	fmt.Printf("Workspaces in %q:\n\n", cfg.CacheDir())
	for _, name := range wsps {
		fmt.Println("\t" + formatWsp(m, current, name))
	}
	fmt.Printf("\nCurrent workspace is marked with ' %s '.\n", defMark)
}

func formatWsp(m manager, current string, name string) string {
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

func printBare(_ manager, current string, workspaces []string) {
	for _, name := range workspaces {
		if current == name {
			fmt.Print("*")
		}
		fmt.Println(name)
	}
}
