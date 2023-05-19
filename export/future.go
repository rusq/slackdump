package export

import (
	"context"
	"time"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/types"
	"github.com/slack-go/slack"
)

// this file is named future.go to avoid conflicts when merging with the v3.0.0

// dumper interface defines the methods used to pull data from Slack. It is
// implemented by slackdump.SlackDumper, but is also used by the
// slackdump_test package to mock out the Slack API calls.
//
//go:generate mockgen -destination=mock_dumper_test.go -source=future.go -package=export dumper
type dumper interface {
	// GetUsers gets the list of all users from the Slack API.
	GetUsers(ctx context.Context) (types.Users, error)

	// CurrentUserID gets the ID of the user running the tool.
	CurrentUserID() string

	// StreamChannels gets a list of all channels from the Slack API, and
	// streams them to the provided callback.
	StreamChannels(ctx context.Context, chanTypes []string, cb func(ch slack.Channel) error) error

	// Client gets the Slack client being used.
	Client() *slack.Client

	// DumpRaw gets data from the Slack API and returns a Conversation object.
	DumpRaw(ctx context.Context, link string, oldest time.Time, latest time.Time, processFn ...slackdump.ProcessFunc) (*types.Conversation, error)

	// GetChannelMembers gets the list of members for a channel.
	GetChannelMembers(ctx context.Context, channelID string) ([]string, error)
}
