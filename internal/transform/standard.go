package transform

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/rusq/dlog"
	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/chunk/state"
	"github.com/rusq/slackdump/v2/internal/osext"
	"github.com/rusq/slackdump/v2/internal/structures/files"
	"github.com/rusq/slackdump/v2/types"
)

type Standard struct {
	fs             fsadapter.FS
	nameFn         func(*types.Conversation) string
	updateFileLink bool
}

// NewStandard returns a new Standard transformer, nameFn should return the
// filename for a given conversation.  This is the name that the conversation
// will be written to the filesystem.
func NewStandard(fs fsadapter.FS, nameFn func(*types.Conversation) string) *Standard {
	return &Standard{fs: fs, nameFn: nameFn, updateFileLink: true}
}

func (s *Standard) Transform(st *state.State, basePath string) error {
	if st == nil {
		return fmt.Errorf("nil state")
	}
	rsc, err := st.OpenChunks(basePath)
	if err != nil {
		return err
	}
	defer rsc.Close()

	pl, err := chunk.NewPlayer(rsc)
	if err != nil {
		return err
	}

	allCh := pl.AllChannels()
	for _, ch := range allCh {
		conv, err := s.conversation(pl, st, basePath, ch)
		if err != nil {
			return err
		}
		if err := s.saveConversation(conv); err != nil {
			return err
		}
	}

	return nil
}

func (s *Standard) conversation(pl *chunk.Player, st *state.State, basePath string, ch string) (*types.Conversation, error) {
	mm, err := pl.AllMessages(ch)
	if err != nil {
		return nil, err
	}
	conv := &types.Conversation{
		ID:       ch,
		Messages: make([]types.Message, 0, len(mm)),
	}
	for i := range mm {
		if mm[i].SubType == "thread_broadcast" {
			// this we don't eat.
			// skip thread broadcasts, they're not useful
			continue
		}
		var sdm types.Message
		sdm.Message = mm[i]
		if mm[i].ThreadTimestamp != "" {
			// if there's a thread timestamp, we need to find and add it.
			thread, err := pl.AllThreadMessages(ch, mm[i].ThreadTimestamp)
			if err != nil {
				return nil, err
			}
			sdm.ThreadReplies = types.ConvertMsgs(thread)
			// update the file links, if requested
			if err := s.transferFiles(st, basePath, sdm.ThreadReplies, ch); err != nil {
				return nil, err
			}
		}
		conv.Messages = append(conv.Messages, sdm)
	}
	// update the file links, if requested
	if err := s.transferFiles(st, basePath, conv.Messages, ch); err != nil {
		return nil, err
	}
	return conv, nil
}

func (s *Standard) transferFiles(st *state.State, basePath string, mm []types.Message, ch string) error {
	for i := range mm {
		if mm[i].Files == nil {
			continue
		}
		for j := range mm[i].Files {
			fp := st.FilePath(ch, mm[i].Files[j].ID)
			if fp == "" {
				return fmt.Errorf("unable to generate the filename for: %v", mm[i].Files[j])
			}
			srcPath := filepath.Join(basePath, fp)
			fsTrgPath := filepath.Join(ch, "attachments", filepath.Base(srcPath))
			if err := osext.MoveFile(srcPath, s.fs, fsTrgPath); err != nil {
				dlog.Printf("file missing: %q", srcPath)
				return fmt.Errorf("error moving %q to %q", srcPath, fsTrgPath)
			}
			// TODO: simplify this
			if s.updateFileLink {
				if err := files.UpdateFileLinksAll(&mm[i].Files[j], func(ptrS *string) error {
					*ptrS = fsTrgPath
					return nil
				}); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *Standard) saveConversation(conv *types.Conversation) error {
	if conv == nil {
		return fmt.Errorf("nil conversation")
	}
	f, err := s.fs.Create(s.nameFn(conv))
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(conv)
}
