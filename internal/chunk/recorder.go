package chunk

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk/state"
	"github.com/rusq/slackdump/v3/internal/osext"
)

// Recorder records all the data it receives into a writer.
type Recorder struct {
	mu    sync.Mutex
	enc   Encoder // encoder to use for the chunks
	state *state.State
}

// Encoder is the interface that wraps the Encode method.
type Encoder interface {
	Encode(ctx context.Context, chunk Chunk) error
}

// Option is a function that configures the Recorder.
type Option func(r *Recorder)

// WithEncoder allows you to specify a custom encoder to use for the chunks.
// By default [json.Encoder] is used.
func WithEncoder(enc Encoder) Option {
	return func(r *Recorder) {
		r.enc = enc
	}
}

type jsonEncoder struct {
	enc *json.Encoder
}

func (j *jsonEncoder) Encode(ctx context.Context, chunk Chunk) error {
	return j.enc.Encode(chunk)
}

// NewRecorder creates a new recorder to writer.
func NewRecorder(w io.Writer, options ...Option) *Recorder {
	filename := "unknown"
	if f, ok := w.(osext.Namer); ok {
		filename = f.Name()
	}
	rec := &Recorder{
		enc:   &jsonEncoder{json.NewEncoder(w)},
		state: state.New(filename),
	}
	for _, opt := range options {
		opt(rec)
	}
	return rec
}

// NewCustomRecorder creates a new recorder with a custom encoder.
func NewCustomRecorder(name string, enc Encoder, options ...Option) *Recorder {
	rec := &Recorder{
		enc:   enc,
		state: state.New(name),
	}
	for _, opt := range options {
		opt(rec)
	}
	return rec
}

// Messages is called for each message chunk that is retrieved.
func (rec *Recorder) Messages(ctx context.Context, channelID string, numThreads int, isLast bool, m []slack.Message) error {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	chunk := Chunk{
		Type:       CMessages,
		Timestamp:  time.Now().UnixNano(),
		ChannelID:  channelID,
		IsLast:     isLast,
		Count:      len(m),
		NumThreads: numThreads,
		Messages:   m,
	}
	if err := rec.enc.Encode(ctx, chunk); err != nil {
		return err
	}
	for i := range m {
		rec.state.AddMessage(channelID, m[i].Timestamp)
	}
	return nil
}

// Files is called for each file chunk that is retrieved. The parent message is
// passed in as well.
func (rec *Recorder) Files(ctx context.Context, channel *slack.Channel, parent slack.Message, f []slack.File) error {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	chunk := Chunk{
		Type:      CFiles,
		Timestamp: time.Now().UnixNano(),
		ChannelID: channel.ID,
		Channel:   channel,
		Parent:    &parent,
		ThreadTS:  parent.ThreadTimestamp,
		Count:     len(f),
		Files:     f,
	}
	if err := rec.enc.Encode(ctx, chunk); err != nil {
		return err
	}
	for i := range f {
		rec.state.AddFile(channel.ID, f[i].ID, "")
	}
	return nil
}

// ThreadMessages is called for each of the thread messages that are
// retrieved. The parent message is passed in as well.
func (rec *Recorder) ThreadMessages(ctx context.Context, channelID string, parent slack.Message, threadOnly, isLast bool, tm []slack.Message) error {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	chunks := Chunk{
		Type:       CThreadMessages,
		Timestamp:  time.Now().UnixNano(),
		ChannelID:  channelID,
		Parent:     &parent,
		ThreadTS:   parent.ThreadTimestamp,
		ThreadOnly: threadOnly,
		IsLast:     isLast,
		Count:      len(tm),
		Messages:   tm,
	}
	if err := rec.enc.Encode(ctx, chunks); err != nil {
		return err
	}
	for i := range tm {
		rec.state.AddThread(channelID, parent.ThreadTimestamp, tm[i].Timestamp)
	}
	return nil
}

// ChannelInfo records a channel information.  threadTS should be set to
// threadTS, if ChannelInfo is called while streaming a thread (user requested
// a thread).
func (rec *Recorder) ChannelInfo(ctx context.Context, channel *slack.Channel, threadTS string) error {
	rec.mu.Lock()
	defer rec.mu.Unlock()

	chunk := Chunk{
		Type:      CChannelInfo,
		Timestamp: time.Now().UnixNano(),
		ChannelID: channel.ID,
		ThreadTS:  threadTS,
		Channel:   channel,
	}
	if err := rec.enc.Encode(ctx, chunk); err != nil {
		return err
	}
	rec.state.AddChannel(channel.ID)
	return nil
}

// Users records a slice of users.
func (rec *Recorder) Users(ctx context.Context, users []slack.User) error {
	chunk := Chunk{
		Type:      CUsers,
		Timestamp: time.Now().UnixNano(),
		Count:     len(users),
		Users:     users,
	}
	if err := rec.enc.Encode(ctx, chunk); err != nil {
		return err
	}
	return nil
}

// Channel records a slice of channels.
func (rec *Recorder) Channels(ctx context.Context, channels []slack.Channel) error {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	chunk := Chunk{
		Type:      CChannels,
		Timestamp: time.Now().UnixNano(),
		Count:     len(channels),
		Channels:  channels,
	}
	if err := rec.enc.Encode(ctx, chunk); err != nil {
		return err
	}
	return nil
}

// State returns the current recorder state.
func (rec *Recorder) State() (*state.State, error) {
	rec.mu.Lock()
	defer rec.mu.Unlock()

	return rec.state, nil
}

// Close closes the recorder (it's a noop for now).
func (rec *Recorder) Close() error {
	return nil
}

// WorkspaceInfo is called when workspace info is retrieved.
func (rec *Recorder) WorkspaceInfo(ctx context.Context, atr *slack.AuthTestResponse) error {
	rec.mu.Lock()
	defer rec.mu.Unlock()

	chunk := Chunk{
		Type:          CWorkspaceInfo,
		Timestamp:     time.Now().UnixNano(),
		WorkspaceInfo: atr,
	}
	if err := rec.enc.Encode(ctx, chunk); err != nil {
		return err
	}
	return nil
}

// ChannelUsers records the channel users
func (rec *Recorder) ChannelUsers(ctx context.Context, channelID string, threadTS string, users []string) error {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	chunk := Chunk{
		Type:         CChannelUsers,
		ChannelID:    channelID,
		Count:        len(users),
		Timestamp:    time.Now().UnixNano(),
		ChannelUsers: users,
	}
	if err := rec.enc.Encode(ctx, chunk); err != nil {
		return err
	}

	return nil
}

// SearchMessages records the result of a message search.
func (rec *Recorder) SearchMessages(ctx context.Context, query string, sm []slack.SearchMessage) error {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	chunk := Chunk{
		Type:           CSearchMessages,
		Timestamp:      time.Now().UnixNano(),
		Count:          len(sm),
		SearchQuery:    query,
		SearchMessages: sm,
	}
	if err := rec.enc.Encode(ctx, chunk); err != nil {
		return err
	}
	return nil
}

// SearchMessages records the result of a file search.
func (rec *Recorder) SearchFiles(ctx context.Context, query string, sf []slack.File) error {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	chunk := Chunk{
		Type:        CSearchFiles,
		Timestamp:   time.Now().UnixNano(),
		Count:       len(sf),
		SearchQuery: query,
		SearchFiles: sf,
	}
	if err := rec.enc.Encode(ctx, chunk); err != nil {
		return err
	}
	return nil
}
