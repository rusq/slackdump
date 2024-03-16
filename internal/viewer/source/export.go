package source

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/export"
)

// ZIPExport implements viewer.Sourcer for the zip file Slack export format.
type ZIPExport struct {
	z         *zip.ReadCloser
	chanNames map[string]string // maps the channel id to the channel name.
	name      string            // name of the file
}

func NewExport(zipfile string) (*ZIPExport, error) {
	if strings.ToLower(filepath.Ext(zipfile)) != ".zip" {
		return nil, errors.New("not a zip file")
	}
	// init from zip
	rc, err := zip.OpenReader(zipfile)
	if err != nil {
		return nil, err
	}
	z := &ZIPExport{
		z:    rc,
		name: zipfile,
	}

	// initialise channels for quick lookup
	c, err := z.Channels()
	if err != nil {
		return nil, err
	}
	z.chanNames = make(map[string]string, len(c))
	for _, ch := range c {
		z.chanNames[ch.ID] = ch.Name
	}

	return z, nil
}

func (*ZIPExport) findByName(z *zip.ReadCloser, name string) (*zip.File, error) {
	for _, f := range z.File {
		if strings.EqualFold(f.Name, name) {
			return f, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", fs.ErrNotExist, name)
}

func (e *ZIPExport) openFile(f string) (io.ReadCloser, error) {
	zf, err := e.findByName(e.z, f)
	if err != nil {
		return nil, err
	}
	return zf.Open()
}

func (e *ZIPExport) Channels() ([]slack.Channel, error) {
	rc, err := e.openFile("channels.json")
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	var c []slack.Channel
	if err := json.NewDecoder(rc).Decode(&c); err != nil {
		return nil, err
	}
	return c, nil
}

func (e *ZIPExport) Users() ([]slack.User, error) {
	rc, err := e.openFile("users.json")
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	var u []slack.User
	if err := json.NewDecoder(rc).Decode(&u); err != nil {
		return nil, err
	}
	return u, nil
}

func (e *ZIPExport) Close() error {
	return e.z.Close()
}

func (e *ZIPExport) Name() string {
	return e.name
}

func (e *ZIPExport) AllMessages(channelID string) ([]slack.Message, error) {
	// find the channel
	name, ok := e.chanNames[channelID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", fs.ErrNotExist, channelID)
	}
	var mm []slack.Message
	if err := fs.WalkDir(e.z, name, func(pth string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if path.Ext(pth) != ".json" {
			return nil
		}
		// read the file
		var em []export.ExportMessage
		f, err := e.z.Open(pth)
		if err != nil {
			return err
		}
		defer f.Close()
		dec := json.NewDecoder(f)
		if err := dec.Decode(&em); err != nil {
			return err
		}
		for _, m := range em {
			mm = append(mm, slack.Message{Msg: *m.Msg})
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("AllMessages: walk: %s", err)
	}
	return mm, nil
}

func (e *ZIPExport) AllThreadMessages(channelID, threadID string) ([]slack.Message, error) {
	//TODO implement me
	panic("implement me")
}

func (e *ZIPExport) ChannelInfo(channelID string) (*slack.Channel, error) {
	//TODO implement me
	panic("implement me")
}
