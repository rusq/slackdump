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

package source

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/internal/chunk"
)

const (
	attachmentDir = "attachments"
)

// Storage is the interface for the file storage used by the source types.
type Storage interface {
	// FS should return the filesystem with file attachments.
	FS() fs.FS
	// Type should return the storage type.
	Type() StorageType
	// File should return the path of the file WITHIN the filesystem returned
	// by FS().  If file is not found, it should return fs.ErrNotExist.
	File(id string, name string) (string, error)
	// FilePath should return the path to the file f relative to the root of
	// the Source (i.e., for Mattermost, __uploads/ID/Name.ext).
	FilePath(ch *slack.Channel, f *slack.File) string
}

// unsafeFilenameRe is a regular expression that matches unsafe characters in
// filenames.
var (
	unsafeFilenameRe = regexp.MustCompile(`[<>:"/\\|?*]`)
	reservedNames    = map[string]struct{}{
		"CON": {}, "PRN": {}, "AUX": {}, "NUL": {},
		"COM1": {}, "COM2": {}, "COM3": {}, "COM4": {}, "COM5": {}, "COM6": {}, "COM7": {}, "COM8": {}, "COM9": {},
		"LPT1": {}, "LPT2": {}, "LPT3": {}, "LPT4": {}, "LPT5": {}, "LPT6": {}, "LPT7": {}, "LPT8": {}, "LPT9": {},
	}
)

// SanitizeFilename ensures the filename is safe for all OSes, especially
// Windows.
func SanitizeFilename(name string) string {
	safe := unsafeFilenameRe.ReplaceAllString(name, "_")
	safe = strings.TrimRight(safe, " .")
	base := safe
	if dot := strings.Index(base, "."); dot != -1 {
		base = base[:dot]
	}
	if _, found := reservedNames[strings.ToUpper(base)]; found {
		safe = "_" + safe
	}
	if safe == "" {
		safe = "unnamed_file"
	}
	return safe
}

// MattermostFilepath returns the path to the file within the __uploads
// directory.
func MattermostFilepath(_ *slack.Channel, f *slack.File) string {
	return filepath.Join(chunk.UploadsDir, f.ID, SanitizeFilename(f.Name))
}

// MattermostFilepathWithDir returns the path to the file within the given
// directory, but it follows the mattermost naming pattern.  In most cases
// you don't need to use this function.
func MattermostFilepathWithDir(dir string) func(*slack.Channel, *slack.File) string {
	return func(_ *slack.Channel, f *slack.File) string {
		return path.Join(dir, f.ID, SanitizeFilename(f.Name))
	}
}

// StdFilepath returns the path to the file within the "attachments"
// directory.
func StdFilepath(ci *slack.Channel, f *slack.File) string {
	return path.Join(ExportChanName(ci), "attachments", fmt.Sprintf("%s-%s", f.ID, SanitizeFilename(f.Name)))
}

// DumpFilepath returns the path to the file within the channel directory.
func DumpFilepath(ci *slack.Channel, f *slack.File) string {
	return path.Join(chunk.ToFileID(ci.ID, "", false).String(), f.ID+"-"+SanitizeFilename(f.Name))
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

func (r *STMattermost) Type() StorageType {
	return STmattermost
}

func (r *STMattermost) File(id string, name string) (string, error) {
	// Try sanitized name first
	sanitized := SanitizeFilename(name)
	pth := path.Join(id, sanitized)
	if _, err := fs.Stat(r.fs, pth); err == nil {
		return pth, nil
	}
	// Optionally, try original name for backward compatibility
	pthOrig := path.Join(id, name)
	if _, err := fs.Stat(r.fs, pthOrig); err == nil {
		return pthOrig, nil
	}
	return "", fs.ErrNotExist
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

// OpenStandardStorage returns the resolver for the export's standard storage
// format.
func OpenStandardStorage(rootfs fs.FS) (*STStandard, error) {
	idx, err := buildStdFileIdx(rootfs, ".")
	if err != nil {
		return nil, err
	}
	if len(idx) == 0 {
		return nil, fs.ErrNotExist
	}
	return newStandardStorage(rootfs, idx), nil
}

// newStandardStorage returns the resolver for the standard export storage
// format, given the root filesystem and the index of files.  The index is
// built by the [buildStdFileIdx] function.
func newStandardStorage(rootfs fs.FS, idx map[string]string) *STStandard {
	return &STStandard{fs: rootfs, idx: idx}
}

// buildStdFileIdx walks the fsys, finding all "attachments" subdirectories, and
// indexes files in them.
func buildStdFileIdx(fsys fs.FS, dir string) (map[string]string, error) {
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

func (r *STStandard) Type() StorageType {
	return STstandard
}

func (r *STStandard) FilePath(ci *slack.Channel, f *slack.File) string {
	return StdFilepath(ci, f)
}

func (r *STStandard) File(id string, name string) (string, error) {
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

// NoStorage is the Storage that returns fs.ErrNotExist for all files.
type NoStorage struct{}

func (NoStorage) FS() fs.FS {
	return fakefs{}
}

func (NoStorage) Type() StorageType {
	return STnone
}

func (NoStorage) File(id string, name string) (string, error) {
	return "", fs.ErrNotExist
}

func (NoStorage) FilePath(*slack.Channel, *slack.File) string {
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
		isChan, err := filepath.Match("[CDG]*.json", d.Name())
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
				if errors.Is(err, fs.ErrNotExist) {
					// not all channels may contain files.
					return nil
				}
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

func (r *STDump) Type() StorageType {
	return STdump
}

func (r *STDump) File(id string, name string) (string, error) {
	// Try sanitized name first
	for _, pth := range []string{
		r.idx[id],
	} {
		if pth == "" {
			continue
		}
		if _, err := fs.Stat(r.fs, pth); err == nil {
			return pth, nil
		}
	}
	return "", fs.ErrNotExist
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

func (r *AvatarStorage) Type() StorageType {
	return STAvatar
}

// AvatarParams is a convenience function that returns the user ID and the base
// name of the original avatar filename to be passed to AvatarStorage.File function.
// For example:
//
//	var as *AvatarStorage
//	var u *slack.User
//	fmt.Println(as.File(AvatarParams(u)))
func AvatarParams(u *slack.User) (userID string, filename string) {
	return u.ID, path.Base(u.Profile.ImageOriginal)
}

func (r *AvatarStorage) File(userID string, imageOriginalBase string) (string, error) {
	pth := path.Join(userID, imageOriginalBase)
	_, err := fs.Stat(r.fs, pth)
	if err != nil {
		return "", err
	}
	return pth, nil
}

// FilePath is unused on AvatarStorage.
func (r *AvatarStorage) FilePath(_ *slack.Channel, _ *slack.File) string {
	return ""
}
