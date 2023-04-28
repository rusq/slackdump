package transform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"runtime/trace"

	"github.com/rusq/dlog"
	"github.com/rusq/fsadapter"
	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/chunk/state"
	"github.com/rusq/slackdump/v2/internal/nametmpl"
	"github.com/rusq/slackdump/v2/internal/osext"
	"github.com/rusq/slackdump/v2/internal/structures/files"
	"github.com/rusq/slackdump/v2/types"
)

type Standard struct {
	// fs is the filesystem to write the transformed data to.
	fs fsadapter.FS
	// nameFn returns the filename for a given conversation.
	nameFn func(*types.Conversation) (string, error)
	// updateFileLink indicates whether the file link should be updated
	// with the path to the file within the archive/directory.
	updateFileLink bool
	// seenFiles ensures that if two messages reference the same file
	// the Files method won't be called twice.
	seenFiles map[string]struct{}
}

// StandardOption is a function that configures a Standard transformer.
type StandardOption func(*Standard)

// WithUpdateFileLink allows to specify whether the file link should be
// updated with the path to the file within the archive/directory.
func WithUpdateFileLink(updateFileLink bool) StandardOption {
	return func(s *Standard) {
		s.updateFileLink = updateFileLink
	}
}

// WithNameFn allows to specify a custom function to generate the filename
// for the conversation.  By default the conversation ID is used.
func WithNameFn(nameFn func(*types.Conversation) (string, error)) StandardOption {
	return func(s *Standard) {
		s.nameFn = nameFn
	}
}

// NewStandard returns a new Standard transformer, nameFn should return the
// filename for a given conversation.  This is the name that the conversation
// will be written to the filesystem.
func NewStandard(fs fsadapter.FS, opts ...StandardOption) *Standard {
	t := nametmpl.NewDefault()
	s := &Standard{
		fs:        fs,
		nameFn:    t.Execute,
		seenFiles: make(map[string]struct{}),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Transform transforms the chunk file referenced by a state into a final form.
func (s *Standard) Transform(ctx context.Context, basePath string, st *state.State) error {
	ctx, task := trace.NewTask(ctx, "transform.Standard.Transform")
	defer task.End()

	trace.Logf(ctx, "state", "%v, len(files)=%d", st.ChannelInfos, len(st.Files))

	rsc, err := loadState(st, basePath)
	if err != nil {
		return err
	}
	defer rsc.Close()

	cf, err := chunk.FromReader(rsc)
	if err != nil {
		return err
	}
	allCh := cf.AllChannelIDs()
	for _, ch := range allCh {
		rgn := trace.StartRegion(ctx, "transform.Standard.Transform: "+ch)
		conv, err := s.conversation(cf, st, basePath, ch)
		rgn.End()
		if err != nil {
			return err
		}
		if err := s.saveConversation(conv); err != nil {
			return err
		}
	}
	// save state file inside the filesystem
	if err := st.SaveFSA(s.fs, filepath.Join(st.ChunkFilename+".state")); err != nil {
		return err
	}
	return nil
}

func (s *Standard) conversation(cf *chunk.File, st *state.State, basePath string, chID string) (*types.Conversation, error) {
	ci, err := cf.ChannelInfo(chID)
	if err != nil {
		return nil, err
	}
	mm, err := cf.AllMessages(chID)
	if err != nil {
		return nil, err
	}
	conv := &types.Conversation{
		ID:       chID,
		Name:     ci.Name,
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
			thread, err := cf.AllThreadMessages(chID, mm[i].ThreadTimestamp)
			if err != nil {
				return nil, err
			}
			sdm.ThreadReplies = types.ConvertMsgs(thread)
			// move the thread files into the archive.
			if err := s.transferFiles(st, basePath, sdm.ThreadReplies, chID); err != nil {
				return nil, err
			}
		}
		conv.Messages = append(conv.Messages, sdm)
	}
	// move the files of the main conversation into the archive.
	if err := s.transferFiles(st, basePath, conv.Messages, chID); err != nil {
		return nil, err
	}

	return conv, nil
}

func (s *Standard) transferFiles(st *state.State, basePath string, mm []types.Message, ch string) error {
	if st == nil || len(st.Files) == 0 {
		return nil // nothing to do
	}
	for i := range mm {
		if mm[i].Files == nil {
			continue
		}
		for j := range mm[i].Files {
			fp := st.FilePath(ch, mm[i].Files[j].ID)
			if fp == "" {
				return fmt.Errorf("unable to generate the filename for: %v", mm[i].Files[j])
			}
			if _, ok := s.seenFiles[fp]; ok {
				continue
			} else {
				s.seenFiles[fp] = struct{}{}
			}
			srcPath := filepath.Join(basePath, fp)
			fsTrgPath := filepath.Join(ch, filepath.Base(srcPath))
			if err := osext.MoveFile(srcPath, s.fs, fsTrgPath); err != nil {
				dlog.Printf("file missing: %q", srcPath)
				return fmt.Errorf("error moving %q to %q", srcPath, fsTrgPath)
			}
			// TODO: simplify this
			if s.updateFileLink {
				s.updateFilepath(&mm[i].Files[j], fsTrgPath)
			}
		}
	}
	return nil
}

func (s *Standard) updateFilepath(m *slack.File, fsTrgPath string) {
	_ = files.UpdateFileLinksAll(m, func(ptrS *string) error {
		*ptrS = fsTrgPath
		return nil
	})
}

func (s *Standard) saveConversation(conv *types.Conversation) error {
	if conv == nil {
		return fmt.Errorf("nil conversation")
	}
	name, err := s.nameFn(conv)
	if err != nil {
		return err
	}
	f, err := s.fs.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(conv)
}

func loadState(st *state.State, basePath string) (io.ReadSeekCloser, error) {
	if st == nil {
		return nil, fmt.Errorf("fatal:  nil state")
	}
	if !st.IsComplete {
		return nil, fmt.Errorf("fatal:  incomplete state")
	}
	rsc, err := st.OpenChunks(basePath)
	if err != nil {
		return nil, err
	}
	return rsc, nil
}

type Standard2 struct {
	cd    *chunk.Directory
	fsa   fsadapter.FS
	idsC  chan string
	doneC chan struct{}
	errC  chan error
}

func NewStandard2(fsa fsadapter.FS, dir string) (*Standard2, error) {
	cd, err := chunk.OpenDir(dir)
	if err != nil {
		return nil, err
	}
	std := &Standard2{
		cd:    cd,
		fsa:   fsa,
		idsC:  make(chan string),
		doneC: make(chan struct{}),
		errC:  make(chan error, 1),
	}
	go std.worker()
	return std, nil
}

func (s *Standard2) worker() {
	defer close(s.errC)
	for id := range s.idsC {
		if err := stdConvert(s.fsa, s.cd, id); err != nil {
			dlog.Printf("error converting %q: %v", id, err)
			s.errC <- err
		}
	}
}

// Close closes the transformer.
func (s *Standard2) Close() error {
	close(s.idsC)
	for err := range s.errC {
		if err != nil {
			return err
		}
	}
	return nil
}

// Transform sends the id to worker that runs the transformation.
func (s *Standard2) Transform(ctx context.Context, id string) error {
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

func stdConvert(fsa fsadapter.FS, cd *chunk.Directory, fileID string) error {
	cf, err := cd.Open(fileID)
	if err != nil {
		return err
	}
	defer cf.Close()

	channelID, threadID := chunk.SplitFileID(fileID)
	ci, err := cf.ChannelInfo(channelID)
	if err != nil {
		return err
	}
	// determine if this a thread.
	mm, err := cf.AllMessages(channelID)
	if err != nil {
		if !errors.Is(err, chunk.ErrNotFound) {
			return fmt.Errorf("unexpected error when processing %q: %w", fileID, err)
		}
		// this is a thread.
		mm, err = cf.AllThreadMessages(channelID, threadID)
		if err != nil {
			return err
		}
		// get parent message
		// parent, err := cf.ParentMessage(channelID, threadID) // TODO: implement
		// if err != nil {
		// 	return err
		// }
	}
	conv := &types.Conversation{
		ID:       channelID,
		Name:     ci.Name,
		Messages: make([]types.Message, 0, len(mm)),
	}
	for i := range mm {
		if mm[i].SubType == "thread_broadcast" {
			// this we don't eat.
			// skip thread broadcasts, they're not useful
			continue
		}
		var sdm types.Message // slackdump message
		sdm.Message = mm[i]
		if mm[i].ThreadTimestamp != "" && mm[i].ThreadTimestamp == mm[i].Timestamp { // process thread only for parent messages
			// if there's a thread timestamp, we need to find and add it.
			thread, err := cf.AllThreadMessages(channelID, mm[i].ThreadTimestamp)
			if err != nil {
				return err
			}
			sdm.ThreadReplies = types.ConvertMsgs(thread)
		}
		conv.Messages = append(conv.Messages, sdm)
	}

	f, err := fsa.Create(fileID + ".json")
	if err != nil {
		return fmt.Errorf("fsadapter: unable to create file %s: %w", fileID+".json", err)
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(conv)
}
