package chunk

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"time"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/internal/chunk/state"
)

// Recorder is a special Channeler that records all the data it receives, so
// that it can be replayed later.
type Recorder struct {
	chunks chan Chunk
	errC   chan error

	enc Encoder // encoder to use for the chunks

	state *state.State
}

// Option is a function that configures the Recorder.
type Option func(r *Recorder)

// WithEncoder allows you to specify a custom encoder to use for the chunks.
// By default, json.NewEncoder is used.
func WithEncoder(enc Encoder) Option {
	return func(r *Recorder) {
		r.enc = enc
	}
}

func NewRecorder(w io.Writer, options ...Option) *Recorder {
	filename := "unknown"
	if f, ok := w.(namer); ok {
		filename = f.Name()
	}
	rec := &Recorder{
		chunks: make(chan Chunk),
		errC:   make(chan error, 1),
		enc:    json.NewEncoder(w),
		state:  state.New(filename),
	}
	for _, opt := range options {
		opt(rec)
	}
	go rec.worker(rec.enc)
	return rec
}

// Encoder is the interface that wraps the Encode method.
type Encoder interface {
	Encode(chunk interface{}) error
}

func (rec *Recorder) worker(enc Encoder) {
LOOP:
	for chunk := range rec.chunks {
		if err := enc.Encode(chunk); err != nil {
			select {
			case rec.errC <- err:
				log.Printf("internal error: %s", err)
			default:
				// unable to send, prevent deadlock
				break LOOP
			}
		}
	}
	close(rec.errC)
}

// Messages is called for each message chunk that is retrieved.
func (rec *Recorder) Messages(ctx context.Context, channelID string, numThreads int, isLast bool, m []slack.Message) error {
	select {
	case err := <-rec.errC:
		return err
	case rec.chunks <- Chunk{
		Type:      CMessages,
		Timestamp: time.Now().UnixNano(),
		ChannelID: channelID,
		IsLast:    isLast,
		Count:     len(m),
		Messages:  m,
	}: // ok
		for i := range m {
			rec.state.AddMessage(channelID, m[i].Timestamp)
		}
	}
	return nil
}

// Files is called for each file chunk that is retrieved. The parent message is
// passed in as well.
func (rec *Recorder) Files(ctx context.Context, channelID string, parent slack.Message, isThread bool, f []slack.File) error {
	select {
	case err := <-rec.errC:
		return err
	case rec.chunks <- Chunk{
		Type:      CFiles,
		Timestamp: time.Now().UnixNano(),
		ChannelID: channelID,
		Parent:    &parent,
		IsThread:  isThread,
		Count:     len(f),
		Files:     f,
	}: // ok
		for i := range f {
			rec.state.AddFile(channelID, f[i].ID, "")
		}
	}
	return nil
}

// ThreadMessages is called for each of the thread messages that are
// retrieved. The parent message is passed in as well.
func (rec *Recorder) ThreadMessages(ctx context.Context, channelID string, parent slack.Message, isLast bool, tm []slack.Message) error {
	select {
	case err := <-rec.errC:
		return err
	case rec.chunks <- Chunk{
		Type:      CThreadMessages,
		Timestamp: time.Now().UnixNano(),
		ChannelID: channelID,
		Parent:    &parent,
		IsThread:  true,
		IsLast:    isLast,
		Count:     len(tm),
		Messages:  tm,
	}: // ok
		for i := range tm {
			rec.state.AddThread(channelID, parent.ThreadTimestamp, tm[i].Timestamp)
		}
	}
	return nil
}

// isThread should be set to true, if channelinfo is called while streaming a
// thread (user requested a thread).
func (rec *Recorder) ChannelInfo(ctx context.Context, channel *slack.Channel, isThread bool) error {
	select {
	case err := <-rec.errC:
		return err
	case rec.chunks <- Chunk{
		Type:      CChannelInfo,
		Timestamp: time.Now().UnixNano(),
		ChannelID: channel.ID,
		IsThread:  isThread,
		Channel:   channel,
	}: // ok
		rec.state.AddChannel(channel.ID)
	}
	return nil
}

func (rec *Recorder) Users(ctx context.Context, users []slack.User) error {
	select {
	case err := <-rec.errC:
		return err
	case rec.chunks <- Chunk{
		Type:      CUsers,
		Timestamp: time.Now().UnixNano(),
		Count:     len(users),
		Users:     users,
	}: // ok
	}
	return nil
}

func (rec *Recorder) Channels(ctx context.Context, channels []slack.Channel) error {
	select {
	case err := <-rec.errC:
		return err
	case rec.chunks <- Chunk{
		Type:      CChannels,
		Timestamp: time.Now().UnixNano(),
		Count:     len(channels),
		Channels:  channels,
	}: // ok
	}
	return nil
}

func (rec *Recorder) State() (*state.State, error) {
	return rec.state, nil
}

func (rec *Recorder) Close() error {
	close(rec.chunks)
	return <-rec.errC
}
