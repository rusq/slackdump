package format

import (
	"archive/zip"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func opendump(filename string, archive string) (io.ReadSeekCloser, error) {
	fi, err := detect(filename, archive)
	if err != nil {
		return nil, err
	}
	return fi.open()
}

type fileinfo struct {
	fsys     fs.FS
	filename string
	close    func() error
}

func detect(filename, archive string) (fileinfo, error) {
	isSTDIN := filename == "" || filename == "-"
	if archive != "" && isSTDIN {
		return fileinfo{}, errors.New("archive can't be set if reading from STDIN")
	}
	if archive == "" {
		var fl fileinfo
		if isSTDIN {
			fl = fileinfo{
				fsys:  os.DirFS("."), //doesn't matter
				close: func() error { return nil },
			}
		} else {
			// local file
			fl = fileinfo{
				fsys:     os.DirFS(filepath.Dir(filename)),
				filename: filepath.Base(filename),
				close:    func() error { return nil },
			}
		}
		return fl, nil
	}
	if !strings.HasSuffix(strings.ToLower(archive), ".zip") {
		return fileinfo{}, errors.New("unsupported archive format")
	}
	// a file within the zip archive
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return fileinfo{}, err
	}
	fl := fileinfo{
		fsys:     zr,
		filename: filename,
		close:    zr.Close,
	}
	return fl, nil
}

// open reads the file described by fileinfo and copies it's contents to a
// temporary file to allow seek.
func (fl fileinfo) open() (io.ReadSeekCloser, error) {
	defer fl.close()

	var (
		f   fs.File
		err error
	)
	if fl.IsSTDIN() {
		f = os.Stdin
	} else {
		f, err = fl.fsys.Open(fl.filename)
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tmp, err := os.CreateTemp("", "convert*")
	if err != nil {
		return nil, err
	}
	if n, err := io.Copy(tmp, f); err != nil {
		return nil, err
	} else if n <= 1 {
		return nil, errors.New("invalid file contents")
	}
	if err := tmp.Sync(); err != nil {
		return nil, err
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	return &closeremover{tmp}, nil
}

func (fl fileinfo) IsSTDIN() bool {
	return fl.filename == "" || fl.filename == "-"
}

// closeremover is the type that overrides Close method - in addition to
// closing the underlying file, it also deletes it.
type closeremover struct {
	*os.File
}

func (cr *closeremover) Close() error {
	if err := cr.File.Close(); err != nil {
		return err
	}
	return os.Remove(cr.File.Name())
}
