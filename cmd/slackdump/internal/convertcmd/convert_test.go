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

package convertcmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/internal/fixtures"
)

func TestDatafmtSet_HTML(t *testing.T) {
	var got datafmt
	if err := got.Set("html"); err != nil {
		t.Fatalf("Set(html) error = %v", err)
	}
	if got != Fhtml {
		t.Fatalf("Set(html) = %v, want %v", got, Fhtml)
	}
	if _, ok := converters[Fhtml]; !ok {
		t.Fatal("html converter is not registered")
	}
}

func TestRunConvert_HTML(t *testing.T) {
	t.Run("writes static site to directory", func(t *testing.T) {
		src := writeDumpFixture(t)
		output := filepath.Join(t.TempDir(), "site")
		setConvertTestGlobals(t, output, Fhtml)

		if err := runConvert(t.Context(), CmdConvert, []string{src}); err != nil {
			t.Fatalf("runConvert() error = %v", err)
		}

		if _, err := os.Stat(filepath.Join(output, "index.html")); err != nil {
			t.Fatalf("expected index.html to be written: %v", err)
		}
	})

	t.Run("strips zip output to directory", func(t *testing.T) {
		src := writeDumpFixture(t)
		output := filepath.Join(t.TempDir(), "site.zip")
		setConvertTestGlobals(t, output, Fhtml)

		if err := runConvert(t.Context(), CmdConvert, []string{src}); err != nil {
			t.Fatalf("runConvert() error = %v", err)
		}

		if _, err := os.Stat(filepath.Join(filepath.Dir(output), "site", "index.html")); err != nil {
			t.Fatalf("expected stripped output directory to be written: %v", err)
		}
	})

	t.Run("uses normalized default html output name", func(t *testing.T) {
		src := writeDumpFixture(t)
		output := filepath.Join(t.TempDir(), "slackdump_default.zip")
		setConvertTestGlobals(t, output, Fhtml)

		if err := runConvert(t.Context(), CmdConvert, []string{src}); err != nil {
			t.Fatalf("runConvert() error = %v", err)
		}

		if _, err := os.Stat(filepath.Join(filepath.Dir(output), "slackdump_default", "index.html")); err != nil {
			t.Fatalf("expected normalized default output directory to be written: %v", err)
		}
		if _, err := os.Stat(output); !os.IsNotExist(err) {
			t.Fatalf("expected html conversion not to write zip output, got err=%v", err)
		}
	})
}

func TestNormalizeOutput(t *testing.T) {
	tests := []struct {
		name   string
		format datafmt
		output string
		want   string
	}{
		{name: "chunk strips zip", format: Fchunk, output: "out.zip", want: "out"},
		{name: "database strips zip", format: Fdatabase, output: "out.zip", want: "out"},
		{name: "html strips zip", format: Fhtml, output: "out.zip", want: "out"},
		{name: "export keeps zip", format: Fexport, output: "out.zip", want: "out.zip"},
		{name: "dump keeps zip", format: Fdump, output: "out.zip", want: "out.zip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeOutput(tt.format, tt.output); got != tt.want {
				t.Fatalf("normalizeOutput(%v, %q) = %q, want %q", tt.format, tt.output, got, tt.want)
			}
		})
	}
}

func writeDumpFixture(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "source_dump")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(dir, fixtures.FSTestDumpDir); err != nil {
		t.Fatal(err)
	}
	return dir
}

func setConvertTestGlobals(t *testing.T, output string, format datafmt) {
	t.Helper()
	oldParams := params
	oldOutput := cfg.Output
	oldYesMan := cfg.YesMan
	oldWithFiles := cfg.WithFiles
	oldWithAvatars := cfg.WithAvatars
	t.Cleanup(func() {
		params = oldParams
		cfg.Output = oldOutput
		cfg.YesMan = oldYesMan
		cfg.WithFiles = oldWithFiles
		cfg.WithAvatars = oldWithAvatars
	})

	params.outputfmt = format
	cfg.Output = output
	cfg.YesMan = true
	cfg.WithFiles = false
	cfg.WithAvatars = false
}
