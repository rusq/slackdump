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

package main

import (
	"context"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/progress"
	"github.com/stretchr/testify/require"
)

func Test_loadSecrets(t *testing.T) {
	type args struct {
		files []string
	}
	tests := []struct {
		name    string
		args    args
		setupFn func(t *testing.T, dir string)
		wantEnv map[string]string
	}{
		{
			name: "loads secrets",
			args: args{
				files: []string{".env"},
			},
			setupFn: func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("DOT_ENV=set\n"), 0o666))
			},
			wantEnv: map[string]string{
				"DOT_ENV": "set",
			},
		},
		{
			"loads secrets from multiple files",
			args{
				files: []string{".env", "secrets.txt"},
			},
			func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("DOT_ENV=set\n"), 0o666))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "secrets.txt"), []byte("SECRETS_TXT=set\n"), 0o666))
			},
			map[string]string{
				"DOT_ENV":     "set",
				"SECRETS_TXT": "set",
			},
		},
		{
			"secrets from the second file don't override the first",
			args{
				files: []string{".env", "secrets.txt"},
			},
			func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("DOT_ENV=set\n"), 0o666))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "secrets.txt"), []byte("DOT_ENV=override\nSECRETS_TXT=set"), 0o666))
			},
			map[string]string{
				"DOT_ENV":     "set",
				"SECRETS_TXT": "set",
			},
		},
		{
			"does not override existing environment variables",
			args{
				files: []string{".env", "secrets.txt"},
			},
			func(t *testing.T, dir string) {
				t.Helper()
				t.Setenv("DOT_ENV", "env")
				require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("DOT_ENV=set\n"), 0o666))
			},
			map[string]string{
				"DOT_ENV": "env",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.setupFn != nil {
				tt.setupFn(t, dir)
			}
			for i := range tt.args.files {
				tt.args.files[i] = filepath.Join(dir, tt.args.files[i])
			}
			loadSecrets(tt.args.files)
			for k, v := range tt.wantEnv {
				require.Equal(t, v, os.Getenv(k))
			}
		})
	}
}

// Test_invoke verifies CLI startup without a workspace or live credentials.
func Test_invoke(t *testing.T) {
	if mode := os.Getenv("SLACKDUMP_STARTUP_TEST"); mode != "" {
		var f progress.Flags
		cmd := &base.Command{UsageLine: "slackdump archive", RequireAuth: mode != "run", Run: func(ctx context.Context, _ *base.Command, _ []string) error {
			if mode != "run" {
				t.Fatal("unexpected command execution")
			}
			nested, _, err := f.Begin(ctx)
			if err != nil || nested != nil || len(progress.Options(ctx)) != 1 {
				t.Fatalf("progress reporter was not reused: %v %v", nested, err)
			}
			return nil
		}}
		f.Bind(&cmd.Flag)
		args := []string{"archive", "--workspace", "missing-workspace", "--cache-dir", os.Getenv("TEST_CACHE"), "--trace", os.Getenv("TEST_TRACE")}
		if mode != "quiet" {
			args = append(args, "-v")
		}
		if mode != "disabled" {
			args = append(args, "--progress")
		}
		if mode != "no-file" {
			args = append(args, "--log", os.Getenv("TEST_LOG"))
		}
		err := invoke(cmd, args)
		if mode == "run" {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil {
			t.Fatal("expected missing workspace error")
		}
		base.Exit()
		return
	}
	for _, mode := range []string{"file", "no-file", "disabled", "run", "quiet"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			logfile := filepath.Join(dir, "run.log")
			tracefile := filepath.Join(dir, "trace.out")
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^Test_invoke$")
			command.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + dir, "USERPROFILE=" + dir, "APPDATA=" + filepath.Join(dir, "AppData", "Roaming"), "LOCALAPPDATA=" + filepath.Join(dir, "AppData", "Local"), "XDG_CONFIG_HOME=" + dir, "XDG_CACHE_HOME=" + dir, "NOCOLOR=true", "SLACKDUMP_STARTUP_TEST=" + mode, "TEST_CACHE=" + filepath.Join(dir, "cache"), "TEST_LOG=" + logfile, "TEST_TRACE=" + tracefile}
			if systemRoot := os.Getenv("SystemRoot"); systemRoot != "" {
				command.Env = append(command.Env, "SystemRoot="+systemRoot)
			}
			output, err := command.CombinedOutput()
			if mode == "run" && err != nil {
				t.Fatalf("child: %v: %s", err, output)
			}
			if mode != "run" && err == nil {
				t.Fatalf("expected nonzero child exit: %s", output)
			}
			text := string(output)
			if mode == "disabled" {
				if !strings.Contains(text, "trace will be written to") || !strings.Contains(text, "log messages will be written to file") {
					t.Fatalf("disabled startup changed: %s", text)
				}
				return
			}
			if !strings.Contains(text, "trace will be written to") {
				t.Fatalf("missing original trace announcement: %s", text)
			}
			if mode == "file" || mode == "run" {
				if strings.Count(text, "log messages will be written to file") != 1 {
					t.Fatalf("missing original startup debug: %s", text)
				}
			} else if strings.Contains(text, "log messages will be written to file") || strings.Contains(text, "level=DEBUG") {
				t.Fatalf("unexpected startup debug: %s", text)
			}
			if mode == "run" {
				if !strings.Contains(text, "Runtime") || strings.Count(text, "Channels") != 1 {
					t.Fatalf("missing or duplicated final snapshot: %s", text)
				}
			} else if !strings.Contains(text, "Runtime") {
				t.Fatalf("missing final error snapshot: %s", text)
			}
			if mode == "file" {
				data, err := os.ReadFile(logfile)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(string(data), "trace will be written to") || strings.Contains(string(data), "log messages will be written to file") {
					t.Fatalf("startup diagnostics missing from log: %s", data)
				}
			}
			if _, err := os.Stat(tracefile); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// Test_main exercises the real CLI outcome-reporting boundary with no credentials.
func Test_main(t *testing.T) {
	if mode := os.Getenv("SLACKDUMP_MAIN_TEST"); mode != "" {
		flag.CommandLine = flag.NewFlagSet("slackdump", flag.ContinueOnError)
		args := []string{"slackdump"}
		if mode == "cancel" || mode == "cancel-disabled" {
			var pf progress.Flags
			cmd := &base.Command{UsageLine: "slackdump cancelled", Run: func(context.Context, *base.Command, []string) error { return context.Canceled }}
			pf.Bind(&cmd.Flag)
			base.Slackdump.Commands = []*base.Command{cmd}
			args = append(args, "cancelled")
			if mode == "cancel" {
				args = append(args, "--progress")
			}
		} else {
			args = append(args, "archive", "--workspace", "missing-workspace", "--cache-dir", os.Getenv("TEST_CACHE"), "--log", os.Getenv("TEST_LOG"))
			if mode == "progress" {
				args = append(args, "--progress")
			}
		}
		os.Args = args
		main()
		return
	}
	for _, mode := range []string{"progress", "disabled", "cancel", "cancel-disabled"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			logfile := filepath.Join(dir, "run.log")
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^Test_main$")
			command.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + dir, "USERPROFILE=" + dir, "APPDATA=" + filepath.Join(dir, "AppData", "Roaming"), "LOCALAPPDATA=" + filepath.Join(dir, "AppData", "Local"), "XDG_CONFIG_HOME=" + dir, "XDG_CACHE_HOME=" + dir, "NOCOLOR=true", "SLACKDUMP_MAIN_TEST=" + mode, "TEST_CACHE=" + filepath.Join(dir, "cache"), "TEST_LOG=" + logfile}
			if systemRoot := os.Getenv("SystemRoot"); systemRoot != "" {
				command.Env = append(command.Env, "SystemRoot="+systemRoot)
			}
			output, _ := command.CombinedOutput()
			text := string(output)
			switch mode {
			case "progress":
				if strings.Count(text, "ERROR operation failed") != 1 || strings.Count(text, "Channels") != 1 || !strings.Contains(text, "Runtime") || strings.Contains(text, "missing-workspace") || strings.Index(text, "ERROR operation failed") > strings.Index(text, "Channels") {
					t.Fatalf("error not reported once with final snapshot: %s", text)
				}
				data, err := os.ReadFile(logfile)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Count(string(data), "level=ERROR") != 1 || !strings.Contains(string(data), "missing-workspace") {
					t.Fatalf("full file error not recorded once: %s", data)
				}
			case "disabled":
				if strings.Contains(text, "level=ERROR") {
					t.Fatalf("disabled console routing changed: %s", text)
				}
				data, err := os.ReadFile(logfile)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Count(string(data), "level=ERROR") != 1 || !strings.Contains(string(data), "missing-workspace") || strings.Contains(string(data), "operation failed") {
					t.Fatalf("disabled logfile error formatting changed: %s", data)
				}
			case "cancel":
				if strings.Contains(text, "operation cancelled") || strings.Contains(text, "level=INFO") || strings.Count(text, "Channels") != 1 || !strings.Contains(text, "Runtime") {
					t.Fatalf("cancel leaked info or lost snapshot: %s", text)
				}
			case "cancel-disabled":
				if !strings.Contains(text, "operation cancelled") {
					t.Fatalf("disabled cancellation logging changed: %s", text)
				}
			}
		})
	}
}
