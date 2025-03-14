package convert

import (
	"errors"
	"io"
	"io/fs"
	"log/slog"

	"github.com/rusq/slackdump/v3/internal/convert/transform/fileproc"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/source"
)

type FileCopier struct {
	src      source.Sourcer
	trg      fsadapter.FS
	srcLocFn func(*slack.Channel, *slack.File) string
	trgLocFn func(*slack.Channel, *slack.File) string
	enabled  bool
}

func NewFileCopier(src source.Sourcer, trg fsadapter.FS, srcLoc, trgLoc func(*slack.Channel, *slack.File) string, enabled bool) *FileCopier {
	return &FileCopier{
		src:      src,
		trg:      trg,
		srcLocFn: srcLoc,
		trgLocFn: trgLoc,
		enabled:  enabled,
	}
}

// Copy iterates through the files in the message and copies them to the target
// directory.  Source file location is determined by calling the srcFileLoc
// function, joined with the chunk directory name.  target file location — by
// calling trgFileLoc function, and is relative to the target fsadapter root.
func (c *FileCopier) Copy(ch *slack.Channel, msg *slack.Message) error {
	if !c.enabled {
		return nil
	}
	if msg == nil {
		return errors.New("convert: internal error: copy: nil message")
	} else if len(msg.Files) == 0 {
		// no files to process
		return nil
	}

	var (
		fsys = c.src.Files().FS()
		lg   = slog.With("channel", ch.ID, "ts", msg.Timestamp)
	)
	for _, f := range msg.Files {
		if err := fileproc.IsValidWithReason(&f); err != nil {
			lg.Info("skipping file", "file", f.ID, "error", err)
			continue
		}

		srcpath, err := c.src.Files().File(f.ID, f.Name)
		if err != nil {
			return &copyerror{f.ID, err}
		}
		// srcpath := c.opts.srcFileLoc(ch, &f)
		trgpath := c.trgLocFn(ch, &f)

		sfi, err := fs.Stat(fsys, srcpath)
		if err != nil {
			return &copyerror{f.ID, err}
		}
		if sfi.Size() == 0 {
			lg.Warn("skipping", "file", f.ID, "reason", "empty")
			continue
		}
		lg.Debug("copying", "srcpath", srcpath, "trgpath", trgpath)
		if err := copy2trg(c.trg, trgpath, fsys, srcpath); err != nil {
			return &copyerror{f.ID, err}
		}
	}
	return nil
}

// copy2trg copies the file from the source path to the target path.  Source
// path is absolute, target path is relative to the target FS adapter root.
func copy2trg(trgfs fsadapter.FS, trgpath string, srcfs fs.FS, srcpath string) error {
	in, err := srcfs.Open(srcpath)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := trgfs.Create(trgpath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
