package chunk

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/rusq/slack"
)

func versions(names ...string) ([]int64, error) {
	versions := make([]int64, 0, len(names))
	for _, name := range names {
		_, ver, err := version(name)
		if err != nil {
			return nil, fmt.Errorf("versions: %s: %w", name, err)
		}
		versions = append(versions, ver)
	}
	if len(versions) == 0 {
		return nil, errors.New("no versions found")
	}
	sort.Sort(sort.Reverse(int64s(versions)))
	return versions, nil
}

// version returns the version of the file with the given name. it expects the
// name to be in the format "channels_1612345678.json.gz".
func version(name string) (FileID, int64, error) {
	base := filepath.Base(name)
	// base version
	noExt := strings.TrimSuffix(base, chunkExt)
	if !strings.Contains(base, "_") {
		return FileID(noExt), 0, nil
	}
	id, sVer, found := strings.Cut(noExt, "_")
	if !found {
		return "", 0, fmt.Errorf("version not found in %s", name)
	}
	ver, err := strconv.ParseInt(sVer, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("version: %w", err)
	}
	return FileID(id), ver, nil
}

type versioner[T any] interface {
	// All should return all entities from the file with the given version.
	All(ver int64) ([]T, error)
	// ID should return the unique identifier of the entity.
	ID(T) string
}

type catalogue interface {
	Versions(FileID) ([]int64, error)
}

// latestVer returns the latest versions of the entities from the given file IDs.
func latestVer[T any](d catalogue, v versioner[T], ids ...FileID) ([]T, error) {
	idx := make(map[string]int, 100)
	var all []T
	for _, id := range ids {
		// we expect versions to be sorted in descending order
		versions, err := d.Versions(id)
		if err != nil {
			return nil, err
		}
		if len(versions) == 0 {
			continue
		}
		for _, ver := range versions {
			chunk, err := v.All(ver)
			if err != nil {
				return nil, err
			}
			if len(all) == 0 {
				all = chunk
				for i, el := range all {
					idx[v.ID(el)] = i
				}
			} else {
				updateIdx(all, idx, chunk, v.ID)
			}
		}
	}
	return all, nil
}

// updateIdx updates the index and the all slice with the new chunk of data,
// replacing the existing data if it exists with the newer versions. idfn is a
// function that returns the unique identifier of the element.  It does not
// update the existing data, as it expects versions to be sorted in descending
// order (newest first).
func updateIdx[T any](all []T, idx map[string]int, elements []T, idfn func(T) string) {
	for _, u := range elements {
		id := idfn(u)
		if _, ok := idx[id]; !ok {
			idx[id] = len(all)
			all = append(all, u)
			// as we expect versions to be sorted in descending order, we don't
			// need to update the existing data
		}
	}
}

type userVersion struct {
	Directory versionOpener
}

func (uv *userVersion) All(ver int64) ([]slack.User, error) {
	f, err := uv.Directory.OpenVersion(FUsers, ver)
	if err != nil {
		return nil, fmt.Errorf("unable to open users file: %w", err)
	}
	defer f.Close()
	users, err := f.AllUsers()
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (uv *userVersion) ID(u slack.User) string {
	return u.ID
}

type versionOpener interface {
	OpenVersion(FileID, int64) (*File, error)
}

type workspaceInfoVersion struct {
	Directory versionOpener
}

func (wiv *workspaceInfoVersion) All(ver int64) ([]*slack.AuthTestResponse, error) {
	for _, name := range []FileID{FWorkspace, FUsers, FChannels} {
		f, err := wiv.Directory.OpenVersion(name, ver)
		if err != nil {
			continue
		}
		defer f.Close()
		wi, err := f.WorkspaceInfo()
		if err != nil {
			continue
		}
		return []*slack.AuthTestResponse{wi}, nil
	}
	return nil, errors.New("no workspace info found")
}

func (wiv *workspaceInfoVersion) ID(wi *slack.AuthTestResponse) string {
	return wi.TeamID
}

func filever(id FileID, ver int64) string {
	switch ver {
	case -1:
		return fmt.Sprintf("%s*%s", id, chunkExt)
	case 0:
		return fmt.Sprintf("%s%s", id, chunkExt)
	default:
		return fmt.Sprintf("%s_%d%s", id, ver, chunkExt)
	}
}

// fileversions is a struct that contains information about the file and its
// versions.
type fileversions struct {
	ID FileID
	V  []int64
}

// collectVersions collects all versions of the file chunks in the root of the
// fsys.
func collectVersions(fsys fs.FS) ([]fileversions, error) {
	names, err := fs.Glob(fsys, "*"+chunkExt)
	if err != nil {
		return nil, err
	}
	var fvs []fileversions
	seenIDs := make(map[FileID]struct{})
	for _, name := range names {
		id, _, err := version(name)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", name, err)
		}
		if _, ok := seenIDs[id]; !ok {
			versions, err := allVersions(fsys, id)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", id, err)
			}
			fvs = append(fvs, fileversions{ID: id, V: versions})
			seenIDs[id] = struct{}{}
		}
	}
	return fvs, nil
}

// allVersions returns all versions of the file with the given ID on the
// filesystem fsys.
func allVersions(fsys fs.FS, id FileID) ([]int64, error) {
	names, err := fs.Glob(fsys, filever(id, -1))
	if err != nil {
		return nil, err
	}
	return versions(names...)
}
