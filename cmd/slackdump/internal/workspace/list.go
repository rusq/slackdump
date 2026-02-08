// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
package workspace

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/trace"
	"sort"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/cache"
)

//go:embed assets/list.md
var listMd string

var cmdWspList = &base.Command{
	UsageLine:  baseCommand + " list [flags]",
	Short:      "list saved authentication information",
	Long:       listMd,
	FlagMask:   flagmask,
	PrintFlags: true,
}

var (
	bare = cmdWspList.Flag.Bool("b", false, "bare output format (just names)")
	all  = cmdWspList.Flag.Bool("a", false, "all information, including user")
)

func init() {
	cmdWspList.Run = runList
}

func runList(ctx context.Context, cmd *base.Command, args []string) error {
	m, err := CacheMgr()
	if err != nil {
		base.SetExitStatus(base.SCacheError)
		return err
	}

	// default fmtFn is print full.
	fmtFn := printDefault
	if *bare {
		fmtFn = printBare
	} else if *all {
		fmtFn = printAll
	}

	return list(ctx, m, fmtFn)
}

type formatFunc func(context.Context, io.Writer, manager, string, []string) error

func list(ctx context.Context, m manager, formatter formatFunc) error {
	entries, err := m.List()
	if err != nil {
		if errors.Is(err, cache.ErrNoWorkspaces) {
			base.SetExitStatus(base.SUserError)
			return errors.New("no authenticated workspaces, please run \"" + baseCommand + " new\"")
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

	return formatter(ctx, os.Stdout, m, current, entries)
}

func printDefault(_ context.Context, w io.Writer, m manager, current string, wsps []string) error {
	ew := &errWriter{w: w}
	fmt.Fprintf(ew, "Workspaces in %q:\n\n", cfg.CacheDir())
	for _, row := range simpleList(m, current, wsps) {
		fmt.Fprintf(ew, "%s (file: %s, last modified: %s)\n", row[0], row[1], row[2])
	}
	fmt.Fprintf(ew, "\nCurrent workspace is marked with ' %s '.\n", defMark)
	return ew.Err()
}

func printAll(ctx context.Context, w io.Writer, m manager, current string, wsps []string) error {
	ctx, task := trace.NewTask(ctx, "printAll")
	defer task.End()

	ew := &errWriter{w: w}
	tw := tabwriter.NewWriter(ew, 2, 8, 1, ' ', 0)
	defer tw.Flush()

	fmt.Fprintln(tw, makeHeader(hdrItems...))

	rows := wspInfo(ctx, m, current, wsps)
	for _, row := range rows {
		if _, err := fmt.Fprintln(tw, strings.Join(row, "\t")); err != nil {
			return err
		}
	}
	return ew.Err()
}

func printBare(_ context.Context, w io.Writer, _ manager, current string, workspaces []string) error {
	ew := &errWriter{w: w}
	for _, name := range workspaces {
		if current == name {
			fmt.Fprint(ew, "*")
		}
		fmt.Fprintln(ew, name)
	}
	return ew.Err()
}

func wspInfo(ctx context.Context, m manager, current string, wsps []string) [][]string {
	rows := [][]string{}

	var (
		wg   sync.WaitGroup
		rowC = make(chan []string)
		pool = make(chan struct{}, 8)
	)
	for _, name := range wsps {
		wg.Add(1)
		go func() {
			pool <- struct{}{}
			defer func() {
				<-pool
				wg.Done()
			}()
			rowC <- wspRow(ctx, m, current, name)
		}()
	}
	go func() {
		wg.Wait()
		close(rowC)
	}()
	for row := range rowC {
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i][1] < rows[j][1]
	})
	return rows
}

func wspRow(ctx context.Context, m manager, current, name string) []string {
	curr := ""
	if current == name {
		curr = "*"
	}
	fi, err := m.FileInfo(name)
	if err != nil {
		return []string{curr, name, "", "", "", "", err.Error()}
	}
	info, err := userInfo(ctx, m, name)
	if err != nil {
		return []string{curr, name, fi.Name(), fi.ModTime().Format(timeLayout), "", "", err.Error()}
	}
	return []string{curr, name, fi.Name(), fi.ModTime().Format(timeLayout), info.Team, info.User, "OK"}
}

// userInfo returns the team and user information for the given workspace.
func userInfo(ctx context.Context, m manager, name string) (*slack.AuthTestResponse, error) {
	prov, err := m.LoadProvider(name)
	if err != nil {
		return nil, err
	}
	return prov.Test(ctx)
}
