package source

import (
	"io/fs"
	"path"
	"path/filepath"
	"strings"
)

const (
	mmuploads     = "__uploads"
	attachmentDir = "attachments"
)

// filestorage is the interface for the file storage used by the source types.
type filestorage interface {
	FS() fs.FS
	File(id string, name string) (string, error)
}

// fstMattermost is the filestorage for the mattermost export format.  Files
// are stored in the __uploads subdirectory, and the filestorage is the
// filesystem of the __uploads directory.
//
// Directory structure:
//
//	./__uploads/
//	  +-- <file_id1>/filename.ext
//	  +-- <file_id2>/otherfile.ext
//	  +-- ...
type fstMattermost struct {
	fs fs.FS
}

// newMattermostStorage returns the resolver for the mattermost export format.
// rootfs is the root filesystem of the export.
func newMattermostStorage(rootfs fs.FS) (*fstMattermost, error) {
	// mattermost export format has files in the __uploads subdirectory.
	fsys, err := fs.Sub(rootfs, mmuploads)
	if err != nil {
		return nil, err
	}
	return &fstMattermost{fs: fsys}, nil
}

func (r *fstMattermost) FS() fs.FS {
	return r.fs
}

func (r *fstMattermost) File(id string, name string) (string, error) {
	pth := path.Join(id, name)
	_, err := fs.Stat(r.fs, pth)
	if err != nil {
		return "", err
	}
	return pth, nil
}

// fstStandard is the filestorage for the standard export format.  Files are
// stored in the "attachments" subdirectories, and the filestorage is the
// filesystem of the export.
//
// Directory structure:
//
//	./
//	  +-- <channel_name>/
//	  |   +-- attachments/<file_id1>-filename.ext
//	  |   +-- attachments/<file_id2>-otherfile.ext
//	  |   +-- ...
//	  +-- ...
type fstStandard struct {
	fs  fs.FS
	idx map[string]string
}

func newStandardStorage(rootfs fs.FS, idx map[string]string) *fstStandard {
	return &fstStandard{fs: rootfs, idx: idx}
}

// buildFileIndex walks the fsys, finding all "attachments" subdirectories, and
// indexes files in them.
func buildFileIndex(fsys fs.FS, dir string) (map[string]string, error) {
	idx := make(map[string]string) // maps the file id to the file name
	if err := fs.WalkDir(fsys, dir, func(pth string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() || d.Name() != attachmentDir {
			return nil
		}
		// read the files in the attachment directory
		return fs.WalkDir(fsys, pth, func(pth string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			id, _, found := strings.Cut(d.Name(), "-")
			if !found {
				return nil
			}
			idx[id] = pth
			return nil
		})
	}); err != nil {
		return nil, err
	}
	return idx, nil
}

func (r *fstStandard) FS() fs.FS {
	return r.fs
}

func (r *fstStandard) File(id string, _ string) (string, error) {
	pth, ok := r.idx[id]
	if !ok {
		return "", fs.ErrNotExist
	}
	if _, err := fs.Stat(r.fs, pth); err != nil {
		return "", err
	}
	return pth, nil
}

type fakefs struct{}

func (fakefs) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

// fstNotFound is the filestorage that returns fs.ErrNotExist for all files.
type fstNotFound struct{}

func (fstNotFound) FS() fs.FS {
	return fakefs{}
}

func (fstNotFound) File(id string, name string) (string, error) {
	return "", fs.ErrNotExist
}

// fstDump is the filestorage for the dump format.  Files are stored in the
// directories named after the channel IDs.
//
// Directory structure:
//
//	./
//	  +-- <channel_id1>/
//	  |   +-- <file_id1>-filename.ext
//	  |   +-- <file_id2>-otherfile.ext
//	  |   +-- ...
//	  +-- <channel_id1>.json
//	  +-- <channel_id2>/
//	  |   +-- <file_id3>-filename.ext
//	  |   +-- <file_id4>-otherfile.ext
//	  |   +-- ...
//	  +-- <channel_id2>.json
//	  +-- ...
type fstDump struct {
	fs  fs.FS
	idx map[string]string
}

// newDumpStorage returns the file storage of the slackdumpdump format.  fsys
// is the root of the dump.
func newDumpStorage(fsys fs.FS) (*fstDump, error) {
	idx, err := indexDump(fsys)
	if err != nil {
		return nil, err
	}
	return &fstDump{fs: fsys, idx: idx}, nil
}

// indexDump indexes the files in the dump format.
func indexDump(fsys fs.FS) (map[string]string, error) {
	idx := make(map[string]string)
	// 1. find all json files in the root directory, and use their names as the
	// channel IDs.
	var chans []string
	if err := fs.WalkDir(fsys, ".", func(pth string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(d.Name()) != ".json" {
			return nil
		}
		isChan, err := filepath.Match("[CD]*.json", d.Name())
		if err != nil {
			return err
		}
		if !isChan {
			return nil
		}
		chans = append(chans, strings.TrimSuffix(d.Name(), ".json"))
		return nil
	}); err != nil {
		return nil, err
	}
	// 2. scan the channel directories and find the files in them.
	for _, ch := range chans {
		if err := fs.WalkDir(fsys, ch, func(pth string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			id, _, found := strings.Cut(d.Name(), "-")
			if !found {
				return nil
			}
			idx[id] = pth
			return nil
		}); err != nil {
			return nil, err
		}
	}

	return idx, nil
}

func (r *fstDump) FS() fs.FS {
	return r.fs
}

func (r *fstDump) File(id string, name string) (string, error) {
	pth, ok := r.idx[id]
	if !ok {
		return "", fs.ErrNotExist
	}
	if _, err := fs.Stat(r.fs, pth); err != nil {
		return "", err
	}
	return pth, nil
}
