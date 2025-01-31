package chunk

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// versions returns all versions of the files with the given names.  Files
// should be of the same FileID group, it takes the first file as the common
// file ID and will return an error if any of the other files have a different
// file ID.  It will also return an error if there's a duplicate version for
// the same file ID.
func versions(filenames ...string) ([]int64, error) {
	var (
		versions      = make([]int64, 0, len(filenames))
		seenVersions  = make(map[int64]struct{}, len(filenames))
		commonGroupID FileID
	)
	for _, name := range filenames {
		id, ver, err := version(name)
		if err != nil {
			return nil, fmt.Errorf("versions: %s: %w", name, err)
		}
		if commonGroupID == "" {
			commonGroupID = id
		} else if commonGroupID != id {
			return nil, fmt.Errorf("versions: %s: expected %s, got %s", name, commonGroupID, id)
		}
		if _, ok := seenVersions[ver]; ok {
			return nil, fmt.Errorf("versions: %s: duplicate version %d", name, ver)
		} else {
			seenVersions[ver] = struct{}{}
		}
		versions = append(versions, ver)
	}
	if len(versions) == 0 {
		return nil, errors.New("no versions found")
	}
	sort.Sort(sort.Reverse(int64s(versions)))
	return versions, nil
}

const versionSep = "_"

// version returns the version of the file with the given name. it expects the
// name to be in the format "channels_1612345678.json.gz".
func version(name string) (FileID, int64, error) {
	if !strings.HasSuffix(name, chunkExt) {
		return "", 0, fmt.Errorf("expected %s to have extension %s", name, chunkExt)
	}
	base := filepath.Base(name)
	// base version
	noExt := strings.TrimSuffix(base, chunkExt)
	if !strings.Contains(base, versionSep) {
		return FileID(noExt), 0, nil
	}
	id, sVer, found := strings.Cut(noExt, versionSep)
	if !found {
		return "", 0, fmt.Errorf("version not found in %s", name)
	}
	ver, err := strconv.ParseInt(sVer, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("version: %w", err)
	}
	return FileID(id), ver, nil
}

// FileGroup is a struct that contains information about a group of files
// having the same FileID and different versions.
type FileGroup struct {
	ID FileID
	V  []int64
}

// collectGroups collects all versions of the file chunks in the root of the
// fsys.
func collectGroups(fsys fs.FS) ([]FileGroup, error) {
	names, err := fs.Glob(fsys, "*"+chunkExt)
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, fs.ErrNotExist
	}
	var fvs []FileGroup
	seenIDs := make(map[FileID]struct{})
	for _, name := range names {
		id, _, err := version(name)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", name, err)
		}
		if _, ok := seenIDs[id]; !ok {
			versions, err := AllVersions(fsys, id)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", id, err)
			}
			fvs = append(fvs, FileGroup{ID: id, V: versions})
			seenIDs[id] = struct{}{}
		}
	}
	return fvs, nil
}

func walkGroup(fsys fs.FS, fn func(gid FileGroup, err error) error) error {
	fvs, err := collectGroups(fsys)
	if err != nil {
		return err
	}
	for _, fv := range fvs {
		if err := fn(fv, nil); err != nil {
			return err
		}
	}
	return nil
}

// AllVersions returns all versions of the file with the given ID on the
// filesystem fsys.
func AllVersions(fsys fs.FS, id FileID) ([]int64, error) {
	names, err := fs.Glob(fsys, filever(id, -1))
	if err != nil {
		return nil, err
	}
	return versions(names...)
}
