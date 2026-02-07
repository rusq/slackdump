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
package chunk

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/rusq/slack"
)

// Recorder records all the data it receives into a writer.
type Recorder struct {
	mu  sync.Mutex
	enc Encoder // encoder to use for the chunks
}

// Encoder is the interface that wraps the Encode method.
//
//go:generate mockgen -destination=mock_chunk/mock_encoder.go . Encoder
type Encoder interface {
	Encode(ctx context.Context, chunk *Chunk) error
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

func (j *jsonEncoder) Encode(ctx context.Context, chunk *Chunk) error {
	return j.enc.Encode(chunk)
}

// NewRecorder creates a new recorder to writer.
func NewRecorder(w io.Writer, options ...Option) *Recorder {
	rec := &Recorder{
		enc: &jsonEncoder{json.NewEncoder(w)},
	}
	for _, opt := range options {
		opt(rec)
	}
	return rec
}

// NewCustomRecorder creates a new recorder with a custom encoder.
func NewCustomRecorder(enc Encoder, options ...Option) *Recorder {
	rec := &Recorder{
		enc: enc,
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
		Count:      int32(len(m)),
		NumThreads: int32(numThreads),
		Messages:   m,
	}
	if err := rec.enc.Encode(ctx, &chunk); err != nil {
		return err
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
		Count:     int32(len(f)),
		Files:     f,
	}
	if err := rec.enc.Encode(ctx, &chunk); err != nil {
		return err
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
		Count:      int32(len(tm)),
		Messages:   tm,
	}
	if err := rec.enc.Encode(ctx, &chunks); err != nil {
		return err
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
	if err := rec.enc.Encode(ctx, &chunk); err != nil {
		return err
	}
	return nil
}

// Users records a slice of users.
func (rec *Recorder) Users(ctx context.Context, users []slack.User) error {
	chunk := Chunk{
		Type:      CUsers,
		Timestamp: time.Now().UnixNano(),
		Count:     int32(len(users)),
		Users:     users,
	}
	if err := rec.enc.Encode(ctx, &chunk); err != nil {
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
		Count:     int32(len(channels)),
		Channels:  channels,
	}
	if err := rec.enc.Encode(ctx, &chunk); err != nil {
		return err
	}
	return nil
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
	if err := rec.enc.Encode(ctx, &chunk); err != nil {
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
		Count:        int32(len(users)),
		Timestamp:    time.Now().UnixNano(),
		ChannelUsers: users,
	}
	if err := rec.enc.Encode(ctx, &chunk); err != nil {
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
		Count:          int32(len(sm)),
		SearchQuery:    query,
		SearchMessages: sm,
	}
	if err := rec.enc.Encode(ctx, &chunk); err != nil {
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
		Count:       int32(len(sf)),
		SearchQuery: query,
		SearchFiles: sf,
	}
	if err := rec.enc.Encode(ctx, &chunk); err != nil {
		return err
	}
	return nil
}
