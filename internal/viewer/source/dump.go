package source

import (
	"io/fs"
	"os"
	"path"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/types"
)

type Dump struct {
	c    []slack.Channel
	fs   fs.FS
	name string
}

func NewDump(fsys fs.FS, name string) (*Dump, error) {
	d := &Dump{
		fs:   fsys,
		name: name,
	}
	// initialise channels for quick lookup
	c, err := d.Channels()
	if err != nil {
		return nil, err
	}
	d.c = c
	return d, nil
}

func (d Dump) Name() string {
	return d.name
}

func (d Dump) Channels() ([]slack.Channel, error) {
	// if user was diligent enough to dump channels and save them in a file,
	// we can use that.
	if cc, err := unmarshal[[]slack.Channel](d.fs, "channels.json"); err == nil {
		return cc, nil
	}
	// this is highly inefficient: walking all files, reading their contents
	// and finding the channel names.  It is better to have a separate file
	// with the channel names and IDs.
	var cc []slack.Channel
	if err := fs.WalkDir(d.fs, ".", func(pth string, de fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if de.IsDir() || path.Ext(de.Name()) != ".json" {
			return nil
		}
		c, err := unmarshalOne[types.Conversation](d.fs, pth)
		if err != nil {
			return err
		}
		cc = append(cc, slack.Channel{
			GroupConversation: slack.GroupConversation{
				Conversation: slack.Conversation{
					ID: c.ID,
				},
				Name: structures.NVL(c.Name, c.ID), // dump files do not have channel names for private conversations.
			},
		})
		return nil
	}); err != nil {
		return nil, err
	}
	return cc, nil
}

func (d Dump) Users() ([]slack.User, error) {
	u, err := unmarshal[[]slack.User](d.fs, "users.json")
	if err != nil {
		if os.IsNotExist(err) {
			return []slack.User{}, nil // user db not available
		}
		return nil, err
	}
	return u, nil
}

func (d Dump) AllMessages(channelID string) ([]slack.Message, error) {
	c, err := unmarshalOne[types.Conversation](d.fs, channelID+".json")
	if err != nil {
		return nil, err
	}
	return convertMessages(c.Messages), nil
}

func convertMessages(cm []types.Message) []slack.Message {
	var mm = make([]slack.Message, len(cm))
	for i := range cm {
		mm[i] = cm[i].Message
	}
	return mm
}

func (d Dump) AllThreadMessages(channelID, threadID string) ([]slack.Message, error) {
	cm, err := d.findThread(channelID, threadID)
	if err != nil {
		return nil, err
	}
	return convertMessages(cm), nil
}

func (d Dump) findThread(channelID, threadID string) ([]types.Message, error) {
	c, err := unmarshalOne[types.Conversation](d.fs, channelID+".json")
	if err != nil {
		return nil, err
	}
	for _, m := range c.Messages {
		if m.ThreadTimestamp == threadID {
			return m.ThreadReplies, nil
		}
	}
	return nil, fs.ErrNotExist
}

func (d Dump) ChannelInfo(channelID string) (*slack.Channel, error) {
	for _, c := range d.c {
		if c.ID == channelID {
			return &c, nil
		}
	}
	return nil, fs.ErrNotExist
}
