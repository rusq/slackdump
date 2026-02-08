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
package diag

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
)

var cmdUnzip = &base.Command{
	UsageLine: "slackdump tools unzip [flags] <zipfile>",
	Short:     "unzip a zip file",
	Long: `# Unzip tool

Unzip tool is provided as a convenience to extract the contents of a zip file,
for example, a Slack export.

It may be useful if you're getting "Illegal byte sequence" when using the
"unzip" program shipped with your OS, and your system doesn't have "bsdtar" or
"7z" installed.
`,
	PrintFlags:  true,
	CustomFlags: true,
	Run:         runUnzip,
}

var (
	modeList bool
	exDir    string
)

func init() {
	cmdUnzip.Flag.BoolVar(&modeList, "l", false, "list files")
	cmdUnzip.Flag.StringVar(&exDir, "d", "", "extract files into `exdir`")
}

func runUnzip(ctx context.Context, cmd *base.Command, args []string) error {
	if err := cmd.Flag.Parse(args); err != nil {
		base.SetExitStatus(base.SInvalidParameters)
		return err
	}
	if cmd.Flag.NArg() < 1 {
		return errors.New("missing zip file argument")
	}

	zipFile := cmd.Flag.Arg(0)

	var err error
	if modeList {
		err = listZip(os.Stdout, zipFile)
	} else {
		err = unzip(zipFile, exDir)
	}
	if err != nil {
		base.SetExitStatus(base.SGenericError)
		return err
	}
	return nil
}

func listZip(w io.Writer, zipFile string) error {
	zr, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer zr.Close()

	if _, err := fmt.Fprintf(w,
		"  Length      Date    Time    Name\n"+
			"---------  ---------- -----   ----\n",
	); err != nil {
		return err
	}

	var files int
	var size uint64
	for _, f := range zr.File {
		if _, err := fmt.Fprintf(w, "% 9d  %s   %s\n", f.UncompressedSize64, f.Modified.Format("01-02-2006 15:04"), f.Name); err != nil {
			return err
		}
		files++
		size += f.UncompressedSize64
	}

	if _, err := fmt.Fprintf(w,
		"---------                     -------\n"+
			"% 9d                     %d files\n",
		size, files,
	); err != nil {
		return err
	}
	return nil
}

func unzip(zipFile, exDir string) error {
	if exDir != "" {
		if err := os.MkdirAll(exDir, 0o755); err != nil {
			return err
		}
	} else {
		exDir = "."
	}
	zr, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer zr.Close()

	if err := os.CopyFS(exDir, zr); err != nil {
		return err
	}
	log.Printf("unzipped to %q", exDir)

	return nil
}
