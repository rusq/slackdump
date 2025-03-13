package source

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/chunk"
)

const (
	attachmentDir = "attachments"
)

// Storage is the interface for the file storage used by the source types.
type Storage interface {
	// FS should return the filesystem with file attachments.
	FS() fs.FS
	// File should return the path of the file WITHIN the filesystem returned
	// by FS().
	File(id string, name string) (string, error)
	// FilePath should return the path to the file f relative to the root of
	// the Source (i.e. __uploads/ID/Name.ext).
	FilePath(ch *slack.Channel, f *slack.File) string
}

// MattermostFilepath returns the path to the file within the __uploads
// directory.
func MattermostFilepath(_ *slack.Channel, f *slack.File) string {
	return filepath.Join(chunk.UploadsDir, f.ID, f.Name)
}

// MattermostFilepathWithDir returns the path to the file within the given
// directory, but it follows the mattermost naming pattern.
func MattermostFilepathWithDir(dir string) func(*slack.Channel, *slack.File) string {
	return func(_ *slack.Channel, f *slack.File) string {
		return path.Join(dir, f.ID, f.Name)
	}
}

// StdFilepath returns the path to the file within the "attachments"
// directory.
func StdFilepath(ci *slack.Channel, f *slack.File) string {
	return path.Join(ExportChanName(ci), "attachments", fmt.Sprintf("%s-%s", f.ID, f.Name))
}

// DumpFilepath returns the path to the file within the channel directory.
func DumpFilepath(ci *slack.Channel, f *slack.File) string {
	return path.Join(chunk.ToFileID(ci.ID, "", false).String(), f.ID+"-"+f.Name)
}

// STMattermost is the Storage for the mattermost export format.  Files
// are stored in the __uploads subdirectory, and the Storage is the
// filesystem of the __uploads directory.
//
// Directory structure:
//
//	./__uploads/
//	  +-- <file_id1>/filename.ext
//	  +-- <file_id2>/otherfile.ext
//	  +-- ...
type STMattermost struct {
	fs fs.FS
}

// OpenMattermostStorage returns the resolver for the mattermost export format.
// rootfs is the root filesystem of the export.
func OpenMattermostStorage(rootfs fs.FS) (*STMattermost, error) {
	// mattermost export format has files in the __uploads subdirectory.
	if _, err := fs.Stat(rootfs, chunk.UploadsDir); err != nil {
		return nil, err
	}
	fsys, err := fs.Sub(rootfs, chunk.UploadsDir)
	if err != nil {
		return nil, err
	}
	return &STMattermost{fs: fsys}, nil
}

func (r *STMattermost) FS() fs.FS {
	return r.fs
}

func (r *STMattermost) File(id string, name string) (string, error) {
	pth := path.Join(id, name)
	_, err := fs.Stat(r.fs, pth)
	if err != nil {
		return "", err
	}
	return pth, nil
}

func (r *STMattermost) FilePath(_ *slack.Channel, f *slack.File) string {
	return MattermostFilepath(nil, f)
}

// STStandard is the Storage for the standard export format.  Files are
// stored in the "attachments" subdirectories, and the Storage is the
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
type STStandard struct {
	fs  fs.FS
	idx map[string]string
}

func OpenStandardStorage(rootfs fs.FS, idx map[string]string) *STStandard {
	return &STStandard{fs: rootfs, idx: idx}
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
	if len(idx) == 0 {
		// no files found.
		return nil, fs.ErrNotExist
	}
	return idx, nil
}

func (r *STStandard) FS() fs.FS {
	return r.fs
}

func (r *STStandard) FilePath(ci *slack.Channel, f *slack.File) string {
	return StdFilepath(ci, f)
}

func (r *STStandard) File(id string, _ string) (string, error) {
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

// fstNotFound is the Storage that returns fs.ErrNotExist for all files.
type fstNotFound struct{}

func (fstNotFound) FS() fs.FS {
	return fakefs{}
}

func (fstNotFound) File(id string, name string) (string, error) {
	return "", fs.ErrNotExist
}

func (fstNotFound) FilePath(*slack.Channel, *slack.File) string {
	return ""
}

// STDump is the Storage for the dump format.  Files are stored in the
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
type STDump struct {
	fs  fs.FS
	idx map[string]string
}

// NewDumpStorage returns the file storage of the slackdumpdump format.  fsys
// is the root of the dump.
func NewDumpStorage(fsys fs.FS) (*STDump, error) {
	idx, err := indexDump(fsys)
	if err != nil {
		return nil, err
	}
	return &STDump{fs: fsys, idx: idx}, nil
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

func (r *STDump) FilePath(ci *slack.Channel, f *slack.File) string {
	return DumpFilepath(ci, f)
}

func (r *STDump) FS() fs.FS {
	return r.fs
}

func (r *STDump) File(id string, name string) (string, error) {
	pth, ok := r.idx[id]
	if !ok {
		return "", fs.ErrNotExist
	}
	if _, err := fs.Stat(r.fs, pth); err != nil {
		return "", err
	}
	return pth, nil
}

type AvatarStorage struct {
	fs fs.FS
}

func NewAvatarStorage(fsys fs.FS) (*AvatarStorage, error) {
	if _, err := fs.Stat(fsys, "__avatars"); err != nil {
		return nil, err
	}
	subfs, err := fs.Sub(fsys, "__avatars")
	if err != nil {
		return nil, err
	}
	return &AvatarStorage{fs: subfs}, nil
}

func (r *AvatarStorage) FS() fs.FS {
	return r.fs
}

func (r *AvatarStorage) File(id string, name string) (string, error) {
	pth := path.Join(id, name)
	_, err := fs.Stat(r.fs, pth)
	if err != nil {
		return "", err
	}
	return pth, nil
}

func (r *AvatarStorage) FilePath(_ *slack.Channel, f *slack.File) string {
	return path.Join(f.ID, f.Name)
}
