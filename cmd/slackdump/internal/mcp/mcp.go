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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
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

//go:embed all:assets/layouts all:assets/skills
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
	layoutOpencode   = "opencode"
	layoutClaudeCode = "claude-code"
	layoutCopilot    = "copilot"
)

var projectLayouts = []string{
	layoutOpencode,
	layoutClaudeCode,
	layoutCopilot,
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
	layoutFS, err := fs.Sub(projectsFS, path.Join("assets", "layouts", layout))
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("fs chdir layout: %w", err)
	}
	skillsFS, err := fs.Sub(projectsFS, "assets/skills")
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("fs chdir skills: %w", err)
	}
	if err := initNewProject(tgtDir, layoutFS, skillsFS); err != nil {
		return err
	}
	lg := cfg.Log
	lg.InfoContext(ctx, "SUCCESS: new project created", "in", tgtDir, "layout", layout)
	return nil
}

// layoutManifest describes which files and skills a project layout installs.
type layoutManifest struct {
	// Files are layout-specific config files copied verbatim.
	Files []manifestFile `json:"files"`
	// Skills are shared skill files assembled from assets/skills/.
	Skills []manifestSkill `json:"skills"`
}

// manifestFile copies src (relative to the layout FS) to dst (relative to the
// target directory).
type manifestFile struct {
	Src string `json:"src"`
	Dst string `json:"dst"`
}

// manifestSkill installs the shared skill named Skill (a subdirectory of
// assets/skills/) to the path Dst relative to the target directory.
type manifestSkill struct {
	Skill string `json:"skill"`
	Dst   string `json:"dst"`
}

// initNewProject creates tgtDir (if needed) and assembles the project from the
// layout manifest, copying layout-specific files from layoutFS and shared
// skills from skillsFS.
func initNewProject(tgtDir string, layoutFS fs.FS, skillsFS fs.FS) error {
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

	manifest, err := readManifest(layoutFS)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("read layout manifest: %w", err)
	}

	// Copy layout-specific files.
	for _, f := range manifest.Files {
		if err := copyFSFile(layoutFS, f.Src, tgtDir, f.Dst); err != nil {
			base.SetExitStatus(base.SApplicationError)
			return fmt.Errorf("copy layout file %q: %w", f.Src, err)
		}
	}

	// Copy shared skills.
	for _, s := range manifest.Skills {
		skillFile := path.Join(s.Skill, "SKILL.md")
		if err := copyFSFile(skillsFS, skillFile, tgtDir, s.Dst); err != nil {
			base.SetExitStatus(base.SApplicationError)
			return fmt.Errorf("copy skill %q: %w", s.Skill, err)
		}
	}

	return nil
}

// readManifest reads and parses layout.json from layoutFS.
func readManifest(layoutFS fs.FS) (layoutManifest, error) {
	f, err := layoutFS.Open("layout.json")
	if err != nil {
		return layoutManifest{}, fmt.Errorf("open layout.json: %w", err)
	}
	defer f.Close()

	var m layoutManifest
	if err := json.NewDecoder(f).Decode(&m); err != nil {
		return layoutManifest{}, fmt.Errorf("decode layout.json: %w", err)
	}
	return m, nil
}

// copyFSFile copies the file at srcPath in srcFS to dstPath (relative to
// dstDir), creating parent directories as needed.
func copyFSFile(srcFS fs.FS, srcPath string, dstDir string, dstPath string) error {
	src, err := srcFS.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open %q: %w", srcPath, err)
	}
	defer src.Close()

	abs := filepath.Join(dstDir, filepath.FromSlash(dstPath))
	if err := os.MkdirAll(filepath.Dir(abs), 0o777); err != nil {
		return fmt.Errorf("mkdir for %q: %w", abs, err)
	}

	dst, err := os.OpenFile(abs, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o666)
	if err != nil {
		return fmt.Errorf("create %q: %w", abs, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("write %q: %w", abs, err)
	}
	return nil
}
