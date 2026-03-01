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

// Package mcp contains the CLI command for starting the Slackdump MCP server.
package mcp

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	internalmcp "github.com/rusq/slackdump/v4/internal/mcp"
	"github.com/rusq/slackdump/v4/internal/osext"
	"github.com/rusq/slackdump/v4/source"
)

//go:embed assets/mcp.md
var mdMCP string

//go:embed all:assets/layouts/*
var projectsFS embed.FS

// CmdMCP is the "slackdump mcp" command.
var CmdMCP = &base.Command{
	UsageLine:   "slackdump mcp [flags] [<archive>]",
	Short:       "Start a local MCP server for an archive",
	Long:        mdMCP,
	FlagMask:    cfg.OmitAll,
	PrintFlags:  true,
	RequireAuth: false,
	Run:         runMCP,
}

var (
	listenAddr       string
	transport        string
	newProjectLayout string
)

const (
	layoutOpencode = "opencode"
)

var projectLayouts = []string{
	layoutOpencode,
}

func init() {
	CmdMCP.Flag.StringVar(&transport, "transport", "stdio", "MCP transport: \"stdio\" or \"http\"")
	CmdMCP.Flag.StringVar(&listenAddr, "listen", "127.0.0.1:8483", "address to listen on when -transport=http")
	CmdMCP.Flag.StringVar(&newProjectLayout, "new", "", fmt.Sprintf("creates new project layout for AI. Type may be one of: %v", projectLayouts))
}

func runMCP(ctx context.Context, cmd *base.Command, args []string) error {
	if newProjectLayout != "" {
		if len(args) == 0 {
			base.SetExitStatus(base.SInvalidParameters)
			return errors.New("target directory must be provided (will be created)")
		}
		return runMCPNewProject(ctx, newProjectLayout, args[0])
	}
	return runMCPServer(ctx, cmd, args)
}

func runMCPServer(ctx context.Context, cmd *base.Command, args []string) error {
	lg := cfg.Log

	var mcpOpts []internalmcp.Option
	mcpOpts = append(mcpOpts, internalmcp.WithLogger(lg))

	if len(args) >= 1 {
		archivePath := args[0]
		lg.InfoContext(ctx, "mcp: opening archive", "path", archivePath)

		src, err := source.Load(ctx, archivePath)
		if err != nil {
			base.SetExitStatus(base.SUserError)
			return fmt.Errorf("mcp: open archive: %w", err)
		}
		defer src.Close()

		mcpOpts = append(mcpOpts, internalmcp.WithSource(src))
	} else {
		lg.InfoContext(ctx, "mcp: no archive specified; agent must call load_source before using data tools")
	}

	srv := internalmcp.New(mcpOpts...)

	// Add the command_help tool at the CLI layer because it needs access to
	// cmd/slackdump/internal packages which are forbidden from internal/mcp.
	srv.AddTool(toolCommandHelp())

	switch strings.ToLower(transport) {
	case "stdio", "":
		return srv.ServeStdio(ctx)
	case "http":
		lg.InfoContext(ctx, "mcp: http transport", "addr", listenAddr)
		return srv.ServeHTTP(ctx, listenAddr)
	default:
		base.SetExitStatus(base.SInvalidParameters)
		return fmt.Errorf("mcp: unknown transport %q (use \"stdio\" or \"http\")", transport)
	}
}

func runMCPNewProject(ctx context.Context, layout string, tgtDir string) error {
	// ensure we know the project type before accessing the FS
	if !slices.Contains(projectLayouts, layout) {
		base.SetExitStatus(base.SInvalidParameters)
		return fmt.Errorf("unknown project layout %q. Use one of %v", layout, projectLayouts)
	}
	subfs, err := fs.Sub(projectsFS, path.Join("assets", "layouts", layout))
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("fs chdir: %w", err)
	}
	if err := initNewProject(tgtDir, subfs); err != nil {
		return err
	}
	lg := cfg.Log
	lg.InfoContext(ctx, "SUCCESS: new project created", "in", tgtDir, "layout", layout)
	return nil
}

func initNewProject(tgtDir string, fsys fs.FS) error {
	if err := osext.DirExists(tgtDir); err != nil {
		if errors.Is(err, osext.ErrNotADir) {
			base.SetExitStatus(base.SUserError)
			return fmt.Errorf("%s: %w", tgtDir, err)
		}
		// try creating the dir
		if err := os.MkdirAll(tgtDir, 0o777); err != nil {
			base.SetExitStatus(base.SApplicationError)
			return fmt.Errorf("unable to initialise new project in %q: %w", tgtDir, err)
		}
	}
	if err := os.CopyFS(tgtDir, fsys); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("copy project files: %w", err)
	}
	return nil
}
