package transform

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rusq/dlog"
	"github.com/rusq/fsadapter"
	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/nametmpl"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/types"
)

type Standard struct {
	cd  *chunk.Directory
	fsa fsadapter.FS
	// updateFileLink indicates whether the file link should be updated
	// with the path to the file within the archive/directory.
	updateFileLink bool

	idsC  chan chunk.FileID
	doneC chan struct{}
	errC  chan error
	tmpl  *nametmpl.Template
}

type StdOption func(*Standard)

func StdWithTemplate(tmpl *nametmpl.Template) StdOption {
	return func(s *Standard) {
		s.tmpl = tmpl
	}
}

// StdWithUpdateFileLink indicates whether the file link should be updated
// with the path to the file within the archive/directory.
func StdWithUpdateFileLink(b bool) StdOption {
	return func(s *Standard) {
		s.updateFileLink = b
	}
}

func NewStandard(fsa fsadapter.FS, dir string, opts ...StdOption) (*Standard, error) {
	cd, err := chunk.OpenDir(dir)
	if err != nil {
		return nil, err
	}
	std := &Standard{
		cd:    cd,
		fsa:   fsa,
		tmpl:  nametmpl.NewDefault(),
		idsC:  make(chan chunk.FileID),
		doneC: make(chan struct{}),
		errC:  make(chan error, 1),
	}
	for _, opt := range opts {
		opt(std)
	}
	go std.worker()
	return std, nil
}

func (s *Standard) worker() {
	defer close(s.errC)
	for id := range s.idsC {
		if err := stdConvert(s.fsa, s.cd, id, s.tmpl); err != nil {
			dlog.Printf("error converting %q: %v", id, err)
			s.errC <- err
		}
	}
}

// Close closes the transformer.
func (s *Standard) Close() error {
	close(s.idsC)
	for err := range s.errC {
		if err != nil {
			return err
		}
	}
	return nil
}

// Transform sends the id to worker that runs the transformation.
func (s *Standard) Transform(ctx context.Context, id chunk.FileID) error {
	select {
	case err := <-s.errC:
		return err
	default:
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.idsC <- id:
		// keep going
	}
	return nil
}

func stdConvert(fsa fsadapter.FS, cd *chunk.Directory, id chunk.FileID, nametmpl *nametmpl.Template) error {
	cf, err := cd.Open(id)
	if err != nil {
		return err
	}
	defer cf.Close()

	// threadTS is only populated on the thread only files.  It is safe to
	// rely on it being non-empty to determine if we need a thread or a
	// conversation.
	channelID, threadID := id.Split()
	ci, err := cf.ChannelInfo(channelID)
	if err != nil {
		return err
	}

	var msgs []types.Message
	if threadID == "" {
		msgs, err = stdConversation(cf, ci)
	} else {
		msgs, err = stdThread(cf, ci, threadID)
	}
	if err != nil {
		return err
	}
	conv := &types.Conversation{
		ID:       ci.ID,
		Name:     ci.Name,
		ThreadTS: threadID,
		Messages: msgs,
	}

	f, err := fsa.Create(nametmpl.Execute(conv) + ".json")
	if err != nil {
		return fmt.Errorf("fsadapter: unable to create file %s: %w", id+".json", err)
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(conv)
}

func stdConversation(cf *chunk.File, ci *slack.Channel) ([]types.Message, error) {
	mm, err := cf.AllMessages(ci.ID)
	if err != nil {
		return nil, err
	}
	msgs := make([]types.Message, 0, len(mm))
	for i := range mm {
		if mm[i].SubType == "thread_broadcast" {
			// this we don't eat.  Skip thread broadcasts.
			continue
		}
		var sdm types.Message // slackdump message
		sdm.Message = mm[i]
		if mm[i].ThreadTimestamp != "" && mm[i].ThreadTimestamp == mm[i].Timestamp && mm[i].LatestReply != structures.NoRepliesLatestReply { // process thread only for parent messages
			// if there's a thread timestamp, we need to find and add it.
			thread, err := cf.AllThreadMessages(ci.ID, mm[i].ThreadTimestamp)
			if err != nil {
				return nil, err
			}
			sdm.ThreadReplies = types.ConvertMsgs(thread)
		}
		msgs = append(msgs, sdm)
	}
	return msgs, nil
}

func stdThread(cf *chunk.File, ci *slack.Channel, threadID string) ([]types.Message, error) {
	// this is a thread.
	mm, err := cf.AllThreadMessages(ci.ID, threadID)
	if err != nil {
		return nil, err
	}
	// get parent message
	parent, err := cf.ThreadParent(ci.ID, threadID) // TODO: implement
	if err != nil {
		return nil, err
	}
	mm = append([]slack.Message{*parent}, mm...)

	return types.ConvertMsgs(mm), nil
}
