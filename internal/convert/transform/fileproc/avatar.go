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

package fileproc

import (
	"context"
	"path"
	"path/filepath"

	"github.com/rusq/slack"
)

type AvatarProc struct {
	dl       Downloader
	filepath func(u *slack.User) string
}

func NewAvatarProc(dl Downloader) AvatarProc {
	return AvatarProc{
		dl:       dl,
		filepath: AvatarPath,
	}
}

func (a AvatarProc) Users(ctx context.Context, users []slack.User) error {
	for _, u := range users {
		if u.Profile.ImageOriginal == "" {
			// skip empty
			continue
		}
		if err := a.dl.Download(a.filepath(&u), a.removeDoubleDots(u.Profile.ImageOriginal)); err != nil {
			return err
		}
	}
	return nil
}

func (a AvatarProc) Close() error {
	a.dl.Stop()
	return nil
}

func AvatarPath(u *slack.User) string {
	filename := path.Base(u.Profile.ImageOriginal)
	return filepath.Join(
		"__avatars",
		u.ID,
		filename,
	)
}

func (AvatarProc) removeDoubleDots(uri string) string {
	urilen := len(uri)
	if urilen == 0 {
		return uri // not our problem
	}
	// take care of double full stop before extension.
	ext := path.Ext(uri)
	extlen := len(ext)
	if extlen == 0 || urilen == extlen {
		return uri // what's going on here?
	}
	// check if there's full stop right before the extension
	if idxLast := urilen - extlen - 1; uri[idxLast] == '.' {
		// strip the full stop
		return uri[0:idxLast] + ext
	}
	return uri
}
