package diag

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
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
