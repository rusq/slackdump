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
		if err := a.dl.Download(a.filepath(&u), u.Profile.ImageOriginal); err != nil {
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
