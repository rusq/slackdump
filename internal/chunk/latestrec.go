package chunk

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/rusq/slack"
)

type versioner[T any] interface {
	// All should return all entities from the file with the given version.
	All(id FileID, ver int64) ([]T, error)
	// ID should return the unique identifier of the entity.
	ID(T) string
}

// latestRec returns the latest versions of the entities from the given file IDs.
func latestRec[T any](fsys fs.FS, v versioner[T], ids ...FileID) ([]T, error) {
	idx := make(map[string]int, 100)
	var all []T
	for _, id := range ids {
		// we expect versions to be sorted in descending order
		versions, err := AllVersions(fsys, id)
		if err != nil {
			return nil, err
		}
		if len(versions) == 0 {
			continue
		}
		for _, ver := range versions {
			elements, err := v.All(id, ver)
			if err != nil {
				return nil, err
			}
			if len(all) == 0 {
				// index the first slice of data
				all = elements
				for i, el := range all {
					idx[v.ID(el)] = i
				}
			} else {
				updateIdx(all, idx, elements, v.ID)
			}
		}
	}
	return all, nil
}

func oneRec[T any](fsys fs.FS, v versioner[T], id FileID) (T, error) {
	var t T
	versions, err := AllVersions(fsys, id)
	if err != nil {
		return t, err
	}
	if len(versions) == 0 {
		return t, fmt.Errorf("no versions found for %s", id)
	}
	// find the first occurrence of the data
	for _, ver := range versions {
		elements, err := v.All(id, ver)
		if err != nil {
			return t, err
		}
		if len(elements) == 0 {
			continue
		}
		return elements[0], nil
	}
	return t, fmt.Errorf("%s: %w", id, ErrNotFound)
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
	Directory Catalogue
}

func (uv *userVersion) All(id FileID, ver int64) ([]slack.User, error) {
	f, err := uv.Directory.OpenVersion(id, ver)
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

type workspaceInfoVersion struct {
	Directory Catalogue
}

func (wiv *workspaceInfoVersion) All(_ FileID, ver int64) ([]*slack.AuthTestResponse, error) {
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

type channelInfoVersion struct {
	Directory Catalogue
}

func (civ *channelInfoVersion) All(id FileID, ver int64) ([]slack.Channel, error) {
	f, err := civ.Directory.OpenVersion(id, ver)
	if err != nil {
		return nil, fmt.Errorf("unable to open channels file: %w", err)
	}
	defer f.Close()
	channels, err := f.AllChannelInfos()
	if err != nil {
		return nil, err
	}
	return channels, nil
}

func (civ *channelInfoVersion) ID(c slack.Channel) string {
	return c.ID
}

type messageVersion struct {
	Directory Catalogue
	ChannelID string
}

func (mv *messageVersion) All(id FileID, ver int64) ([]slack.Message, error) {
	f, err := mv.Directory.OpenVersion(id, ver)
	if err != nil {
		return nil, fmt.Errorf("unable to open messages file: %w", err)
	}
	defer f.Close()
	messages, err := f.AllMessages(mv.ChannelID)
	if err != nil {
		return nil, err
	}
	return messages, nil
}

func (mv *messageVersion) ID(m slack.Message) string {
	return m.Timestamp
}

type parentMessageVersion struct {
	Directory Catalogue
	ChannelID string
	ThreadID  string
}

func (pmv *parentMessageVersion) All(id FileID, ver int64) ([]slack.Message, error) {
	f, err := pmv.Directory.OpenVersion(id, ver)
	if err != nil {
		return nil, fmt.Errorf("unable to open messages file: %w", err)
	}
	defer f.Close()
	message, err := f.ThreadParent(pmv.ChannelID, pmv.ThreadID)
	if err != nil {
		return nil, err
	}
	return []slack.Message{*message}, nil
}

func (pmv *parentMessageVersion) ID(m slack.Message) string {
	return m.Timestamp
}

type threadMessageVersion struct {
	Directory Catalogue
	ChannelID string
	ThreadID  string
}

func (tmv *threadMessageVersion) All(id FileID, ver int64) ([]slack.Message, error) {
	f, err := tmv.Directory.OpenVersion(id, ver)
	if err != nil {
		return nil, fmt.Errorf("unable to open messages file: %w", err)
	}
	defer f.Close()
	messages, err := f.AllThreadMessages(tmv.ChannelID, tmv.ThreadID)
	if err != nil {
		return nil, err
	}
	return messages, nil
}

func (tmv *threadMessageVersion) ID(m slack.Message) string {
	return m.Timestamp
}
