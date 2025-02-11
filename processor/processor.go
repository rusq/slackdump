package processor

import (
	"context"
	"errors"
	"io"

	"github.com/rusq/slack"
)

// Conversations is the interface for conversation fetching with files.
//
//go:generate mockgen -destination ../mocks/mock_processor/mock_processor.go github.com/rusq/slackdump/v3/processor Conversations,Users,Channels,ChannelInformer,Filer
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

// JoinChannels joins multiple Channels processors into one.
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
