package source

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/export"
)

// Export implements viewer.Sourcer for the zip file Slack export format.
type Export struct {
	fs        fs.FS
	chanNames map[string]string // maps the channel id to the channel name.
	name      string            // name of the file
}

func NewExport(fsys fs.FS, name string) (*Export, error) {
	z := &Export{
		fs:   fsys,
		name: name,
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

func (e *Export) Channels() ([]slack.Channel, error) {
	return unmarshal[[]slack.Channel](e.fs, "channels.json")
}

func (e *Export) Users() ([]slack.User, error) {
	return unmarshal[[]slack.User](e.fs, "users.json")
}

func (e *Export) Close() error {
	return nil
}

func (e *Export) Name() string {
	return e.name
}

func (e *Export) AllMessages(channelID string) ([]slack.Message, error) {
	// find the channel
	name, ok := e.chanNames[channelID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", fs.ErrNotExist, channelID)
	}
	var mm []slack.Message
	if err := fs.WalkDir(e.fs, name, func(pth string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || path.Ext(pth) != ".json" {
			return nil
		}
		// read the file
		em, err := unmarshal[[]export.ExportMessage](e.fs, pth)
		if err != nil {
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

func (e *Export) AllThreadMessages(channelID, threadID string) ([]slack.Message, error) {
	m, err := e.AllMessages(channelID)
	if err != nil {
		return nil, err
	}
	var tm []slack.Message
	for _, msg := range m {
		if msg.ThreadTimestamp == threadID {
			tm = append(tm, msg)
		}
	}
	return tm, nil
}

func (e *Export) ChannelInfo(channelID string) (*slack.Channel, error) {
	c, err := e.Channels()
	if err != nil {
		return nil, err
	}
	for _, ch := range c {
		if ch.ID == channelID {
			return &ch, nil
		}
	}
	return nil, fmt.Errorf("%s: %s", "channel not found", channelID)
}

func unmarshal[T ~[]S, S any](fsys fs.FS, name string) (T, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var v T
	if err := json.NewDecoder(f).Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}
