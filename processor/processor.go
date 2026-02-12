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

package processor

import (
	"context"
	"errors"
	"io"

	"github.com/rusq/slack"
)

// Conversations is the interface for conversation fetching with files.
//
//go:generate mockgen -destination ../mocks/mock_processor/mock_processor.go github.com/rusq/slackdump/v4/processor Conversations,Users,Channels,ChannelInformer,Filer,WorkspaceInfo,MessageSearcher,FileSearcher,Searcher,Avatars
type Conversations interface {
	Messenger
	Filer
	ChannelInformer

	io.Closer
}

type ChannelInformer interface {
	// ChannelInfo is called for each channel that is retrieved.  ChannelInfo
	// will be called for each direct thread link, and in this case, threadID
	// will be set to the parent message's timestamp.
	ChannelInfo(ctx context.Context, ci *slack.Channel, threadID string) error
	ChannelUsers(ctx context.Context, channelID string, threadTS string, users []string) error
}

// Messenger is the interface that implements only the message fetching.
type Messenger interface {
	// Messages method is called for each message that is retrieved.
	Messages(ctx context.Context, channelID string, numThreads int, isLast bool, messages []slack.Message) error
	// ThreadMessages method is called for each of the thread messages that are
	// retrieved. The parent message is passed in as well.
	ThreadMessages(ctx context.Context, channelID string, parent slack.Message, threadOnly, isLast bool, replies []slack.Message) error
}

type Filer interface {
	// Files method is called for each file that is retrieved. The parent message is
	// passed in as well.
	Files(ctx context.Context, channel *slack.Channel, parent slack.Message, ff []slack.File) error
	io.Closer
}

type Users interface {
	// Users method is called for each user chunk that is retrieved.
	Users(ctx context.Context, users []slack.User) error
}

type WorkspaceInfo interface {
	WorkspaceInfo(context.Context, *slack.AuthTestResponse) error
}

type Channels interface {
	// Channels is called for each channel chunk that is retrieved.
	Channels(ctx context.Context, channels []slack.Channel) error
}

// MessageSearcher is the interface for searching messages.
type MessageSearcher interface {
	// SearchMessages is called for each message chunk that is retrieved.
	SearchMessages(ctx context.Context, query string, messages []slack.SearchMessage) error
	ChannelInformer
}

// FileSearcher is the interface for searching files.
type FileSearcher interface {
	// SearchFiles is called for each of the file chunks that are retrieved.
	SearchFiles(ctx context.Context, query string, files []slack.File) error
	// Filer is embedded here to allow for the Files method to be called.
	Filer
}

// Searcher is the combined interface for searching messages and files.
type Searcher interface {
	MessageSearcher
	FileSearcher
}

// Avatars is the interface for downloading avatars.
type Avatars interface {
	Users
	io.Closer
}

// JointChannels is a processor that joins multiple Channels processors into
// one.
type JointChannels struct {
	pp []Channels
}

// JoinChannels joins multiple Channels processors into one.  Processors are
// called in the order they are passed in.
func JoinChannels(procs ...Channels) *JointChannels {
	return &JointChannels{pp: procs}
}

func (c *JointChannels) Channels(ctx context.Context, ch []slack.Channel) error {
	for _, p := range c.pp {
		if err := p.Channels(ctx, ch); err != nil {
			return err
		}
	}
	return nil
}

func (c *JointChannels) Close() error {
	return closeall(c.pp)
}

// JointUser is a processor that joins multiple Users processors.
type JointUsers struct {
	pp []Users
}

// JoinUsers joins multiple Users processors into one.
func JoinUsers(procs ...Users) *JointUsers {
	return &JointUsers{pp: procs}
}

func (u *JointUsers) Users(ctx context.Context, users []slack.User) error {
	for _, p := range u.pp {
		if err := p.Users(ctx, users); err != nil {
			return err
		}
	}
	return nil
}

func (u *JointUsers) Close() error {
	return closeall(u.pp)
}

type JointMessengers struct {
	pp []Messenger
}

// JoinMessenger joins multiple Messenger processors into one.
func JoinMessenger(procs ...Messenger) Messenger {
	return &JointMessengers{pp: procs}
}

func (m *JointMessengers) Messages(ctx context.Context, channelID string, numThreads int, isLast bool, messages []slack.Message) error {
	for _, p := range m.pp {
		if err := p.Messages(ctx, channelID, numThreads, isLast, messages); err != nil {
			return err
		}
	}
	return nil
}

func (m *JointMessengers) ThreadMessages(ctx context.Context, channelID string, parent slack.Message, threadOnly, isLast bool, replies []slack.Message) error {
	for _, p := range m.pp {
		if err := p.ThreadMessages(ctx, channelID, parent, threadOnly, isLast, replies); err != nil {
			return err
		}
	}
	return nil
}

// closeall closes all the io.Closer instances in the slice.
func closeall[T any](pp []T) error {
	var errs error
	for i := len(pp) - 1; i >= 0; i-- {
		if closer, ok := any(pp[i]).(io.Closer); ok {
			if err := closer.Close(); err != nil {
				errs = errors.Join(errs, err)
			}
		}
	}
	return errs
}

// JointConversations is a processor that joins multiple processors.
// TODO: It's done in a crude way, maybe there's a more elegant way to do this.
type JointConversations struct {
	// these are executed (b)efore
	bci []ChannelInformer
	bmm []Messenger
	bff []Filer
	// c is the Conversations processor
	c Conversations
	// these are executed (a)fter
	aci []ChannelInformer
	amm []Messenger
	aff []Filer
}

// PrependChannelInformer prepends the ChannelInformer to the Conversations.
func PrependChannelInformer(c Conversations, ci ...ChannelInformer) Conversations {
	return &JointConversations{c: c, bci: ci}
}

// PrependMessenger prepends the Messenger to the Conversations.
func PrependMessenger(c Conversations, mm ...Messenger) Conversations {
	return &JointConversations{c: c, bmm: mm}
}

// PrependFiler prepends the Filer to the Conversations.
func PrependFiler(c Conversations, ff ...Filer) Conversations {
	return &JointConversations{c: c, bff: ff}
}

// PrependChannelInformer prepends the ChannelInformer to the Conversations.
func AppendChannelInformer(c Conversations, ci ...ChannelInformer) Conversations {
	return &JointConversations{c: c, aci: ci}
}

// PrependMessenger prepends the Messenger to the Conversations.
func AppendMessenger(c Conversations, mm ...Messenger) Conversations {
	return &JointConversations{c: c, amm: mm}
}

// PrependFiler prepends the Filer to the Conversations.
func AppendFiler(c Conversations, ff ...Filer) Conversations {
	return &JointConversations{c: c, aff: ff}
}

// Files executes the prepended Files processors and then forwards the call to
// the Conversations processor.
func (w *JointConversations) Files(ctx context.Context, channel *slack.Channel, parent slack.Message, ff []slack.File) error {
	var errs error
	for _, f := range w.bff {
		if err := f.Files(ctx, channel, parent, ff); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	if err := w.c.Files(ctx, channel, parent, ff); err != nil {
		errs = errors.Join(errs, err)
	}
	for _, f := range w.aff {
		if err := f.Files(ctx, channel, parent, ff); err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return errs
}

// ChannelInfo executes the prepended ChannelInfo processors and then forwards
// the call to the Conversations processor.
func (w *JointConversations) ChannelInfo(ctx context.Context, ci *slack.Channel, threadID string) error {
	var errs error
	for _, c := range w.bci {
		if err := c.ChannelInfo(ctx, ci, threadID); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	if err := w.c.ChannelInfo(ctx, ci, threadID); err != nil {
		errs = errors.Join(errs, err)
	}
	for _, c := range w.aci {
		if err := c.ChannelInfo(ctx, ci, threadID); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	return errs
}

// ChannelUsers executes the prepended ChannelUsers processors and then forwards
// the call to the Conversations processor.
func (w *JointConversations) ChannelUsers(ctx context.Context, channelID string, threadTS string, users []string) error {
	var errs error
	for _, c := range w.bci {
		if err := c.ChannelUsers(ctx, channelID, threadTS, users); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	if err := w.c.ChannelUsers(ctx, channelID, threadTS, users); err != nil {
		errs = errors.Join(errs, err)
	}
	for _, c := range w.aci {
		if err := c.ChannelUsers(ctx, channelID, threadTS, users); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	return errs
}

// Messages executes the prepended Messages processors and then forwards the
// call to the Conversations processor.
func (w *JointConversations) Messages(ctx context.Context, channelID string, numThreads int, isLast bool, messages []slack.Message) error {
	var errs error
	for _, m := range w.bmm {
		if err := m.Messages(ctx, channelID, numThreads, isLast, messages); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	if err := w.c.Messages(ctx, channelID, numThreads, isLast, messages); err != nil {
		errs = errors.Join(errs, err)
	}
	for _, m := range w.amm {
		if err := m.Messages(ctx, channelID, numThreads, isLast, messages); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	return errs
}

// ThreadMessages executes the prepended ThreadMessages processors and then
// forwards the call to the Conversations processor.
func (w *JointConversations) ThreadMessages(ctx context.Context, channelID string, parent slack.Message, threadOnly, isLast bool, replies []slack.Message) error {
	var errs error
	for _, m := range w.bmm {
		if err := m.ThreadMessages(ctx, channelID, parent, threadOnly, isLast, replies); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	if err := w.c.ThreadMessages(ctx, channelID, parent, threadOnly, isLast, replies); err != nil {
		errs = errors.Join(errs, err)
	}
	for _, m := range w.amm {
		if err := m.ThreadMessages(ctx, channelID, parent, threadOnly, isLast, replies); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	return errs
}

// Close closes all the io.Closer instances in the slice.
func (w *JointConversations) Close() error {
	var errs error
	if err := closeall(w.bff); err != nil {
		errs = errors.Join(errs, err)
	}
	if err := closeall(w.bci); err != nil {
		errs = errors.Join(errs, err)
	}
	if err := closeall(w.bmm); err != nil {
		errs = errors.Join(errs, err)
	}
	if err := w.c.Close(); err != nil {
		return err
	}
	if err := closeall(w.aff); err != nil {
		errs = errors.Join(errs, err)
	}
	if err := closeall(w.aci); err != nil {
		errs = errors.Join(errs, err)
	}
	if err := closeall(w.amm); err != nil {
		errs = errors.Join(errs, err)
	}

	return errs
}

type JointFilers struct {
	pp []Filer
}

// JoinFilers joins multiple Filer processors into one.
func JoinFilers(procs ...Filer) Filer {
	return &JointFilers{pp: procs}
}

func (f *JointFilers) Files(ctx context.Context, channel *slack.Channel, parent slack.Message, ff []slack.File) error {
	for _, p := range f.pp {
		if err := p.Files(ctx, channel, parent, ff); err != nil {
			return err
		}
	}
	return nil
}

func (f *JointFilers) Close() error {
	return closeall(f.pp)
}
