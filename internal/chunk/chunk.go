package chunk

import (
	"context"
	"fmt"
	"strings"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/fasttime"
)

// ChunkType is the type of chunk that was recorded..
type ChunkType uint8

//go:generate stringer -type=ChunkType -trimprefix=C
const (
	CMessages ChunkType = iota
	CThreadMessages
	CFiles
	CUsers
	CChannels
	CChannelInfo
	CWorkspaceInfo
	CChannelUsers
	CStarredItems
	CBookmarks
	CSearchMessages
	CSearchFiles
)

var ErrUnsupChunkType = fmt.Errorf("unsupported chunk type")

// Chunk is a representation of a single chunk of data retrieved from the API.
// A single API call always produces a single Chunk.
type Chunk struct {
	// header
	// Type is the type of the Chunk
	Type ChunkType `json:"t"`
	// Timestamp when the chunk was recorded.
	Timestamp int64 `json:"ts"`
	// ChannelID that this chunk relates to.
	ChannelID string `json:"id,omitempty"`
	// Count is the count of elements in the chunk, i.e. messages or files.
	Count int32 `json:"n,omitempty"`

	// NumThreads is the number of threads in the message chunk.
	NumThreads int32 `json:"nt,omitempty"`
	// ThreadTS is populated if the chunk contains thread related data.  It
	// is Slack's thread_ts.
	ThreadTS string `json:"r,omitempty"`
	// IsLast is set to true if this is the last chunk for the channel or
	// thread.

	IsLast bool `json:"l,omitempty"`
	// ThreadOnly is set to true if the chunk was generated as a result
	// of thread only scraping.
	ThreadOnly bool `json:"to,omitempty"`

	// Channel contains the channel information.  Within the chunk file, it
	// may not be immediately followed by messages from the channel due to
	// concurrent nature of the calls.
	//
	// Populated by ChannelInfo and Files methods.
	Channel *slack.Channel `json:"ci,omitempty"`

	// ChannelUsers contains the user IDs of the users in the channel.
	ChannelUsers []string `json:"cu,omitempty"` // Populated by ChannelUsers

	// Parent is populated in case the chunk is a thread, or a file. Populated
	// by ThreadMessages and Files methods.
	Parent *slack.Message `json:"p,omitempty"`
	// Messages contains a chunk of messages as returned by the API. Populated
	// by Messages and ThreadMessages methods.
	Messages []slack.Message `json:"m,omitempty"`
	// Files contains a chunk of files as returned by the API. Populated by
	// Files method.
	Files []slack.File `json:"f,omitempty"`

	// Users contains a chunk of users as returned by the API. Populated by
	// Users method.
	Users []slack.User `json:"u,omitempty"`
	// Channels contains a chunk of channels as returned by the API. Populated
	// by Channels method.
	Channels []slack.Channel `json:"ch,omitempty"`
	// WorkspaceInfo contains the workspace information as returned by the
	// API.  Populated by WorkspaceInfo.
	WorkspaceInfo *slack.AuthTestResponse `json:"w,omitempty"`
	// StarredItems contains the starred items.
	StarredItems []slack.StarredItem `json:"st,omitempty"` // Populated by StarredItems
	// Bookmarks contains the bookmarks.
	Bookmarks []slack.Bookmark `json:"b,omitempty"` // Populated by Bookmarks
	// SearchQuery contains the search query.
	SearchQuery string `json:"sq,omitempty"` // Populated by SearchMessages and SearchFiles.
	// SearchMessages contains the search results.
	SearchMessages []slack.SearchMessage `json:"sm,omitempty"` // Populated by SearchMessages
	// SearchFiles contains the search results.
	SearchFiles []slack.File `json:"sf,omitempty"` // Populated by SearchFiles
}

// GroupID is a unique ID for a chunk group.  It is used to group chunks of
// the same type together for indexing purposes.  It may or may not be equal
// to the Slack ID of the entity.
type GroupID string

const (
	userChunkID     GroupID = "lusr" // list users
	channelChunkID  GroupID = "lch"  // list channels
	starredChunkID  GroupID = "ls"   // list starred
	wspInfoChunkID  GroupID = "iw"   // info workspace
	srchMsgChunkID  GroupID = "sm"   // search messages results
	srchFileChunkID GroupID = "sf"   // search file results
)

const (
	threadPrefix    = "t"
	filePrefix      = "f"
	chanInfoPrefix  = "ic"
	bookmarkPrefix  = "lb"
	chanUsersPrefix = "lcu"
)

// Chunk ID categories
const (
	catFile   = 'f'
	catInfo   = 'i'
	catList   = 'l'
	catSearch = 's'
)

// ID returns a Group ID for the chunk.
func (c *Chunk) ID() GroupID {
	switch c.Type {
	case CMessages:
		return GroupID(c.ChannelID)
	case CThreadMessages:
		return threadID(c.ChannelID, c.Parent.ThreadTimestamp)
	case CFiles:
		return id(filePrefix, c.ChannelID, c.Parent.Timestamp)
	case CChannelInfo:
		return channelInfoID(c.ChannelID)
	case CChannelUsers:
		return channelUsersID(c.ChannelID)
	case CUsers:
		return userChunkID // static, one team per chunk file
	case CChannels:
		return channelChunkID // static
	case CWorkspaceInfo:
		return wspInfoChunkID // static
	case CStarredItems:
		return starredChunkID // static
	case CBookmarks:
		return id(bookmarkPrefix, c.ChannelID)
	case CSearchMessages:
		return srchMsgChunkID
	case CSearchFiles:
		return srchFileChunkID
	}
	return GroupID(fmt.Sprintf("<unknown:%s>", c.Type))
}

// asChannelID returns the channel ID from the GroupID.  If the GroupID is not
// a channel ID, it returns false.
func (id GroupID) AsChannelID() (channelID string, ok bool) {
	return string(id), id.IsChannel()
}

func (id GroupID) IsChannel() bool {
	if len(id) < 1 {
		return false
	}
	switch id[0] {
	case 'C', 'D', 'G':
		return true
	default:
		return false
	}
}

func id(prefix string, ids ...string) GroupID {
	return GroupID(prefix + strings.Join(ids, ":"))
}

func threadID(channelID, threadTS string) GroupID {
	return id(threadPrefix, channelID, threadTS)
}

func (id GroupID) IsThread() bool {
	return strings.HasPrefix(string(id), threadPrefix)
}

// ExtractChannelID attempts to extract the channel ID from the GroupID if it
// is a channel or a thread ID.  Otherwise, ok will be false.
func (id GroupID) ExtractChannelID() (channelID string, ok bool) {
	if id.IsThread() {
		channelID, _, ok = id.AsThreadID()
		return
	}
	if id.IsChannel() {
		return id.AsChannelID()
	}
	return "", false
}

// asThreadID returns the channelID and threadTS from the GroupID.  If the
// GroupID is not a thread ID, it returns false.
func (id GroupID) AsThreadID() (channelID, threadTS string, ok bool) {
	if !strings.HasPrefix(string(id), threadPrefix) {
		return "", "", false
	}
	parts := strings.Split(string(id[1:]), ":")
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func channelInfoID(channelID string) GroupID {
	return id(chanInfoPrefix, channelID)
}

func channelUsersID(channelID string) GroupID {
	return id(chanUsersPrefix, channelID)
}

func (c *Chunk) String() string {
	return c.Type.String() + ": " + string(c.ID())
}

// Timestamps returns the timestamps of the messages in the chunk.  For files
// and other types of chunks, it returns ErrUnsupChunkType.
func (c *Chunk) Timestamps() ([]int64, error) {
	switch c.Type {
	case CMessages, CThreadMessages:
		return c.messageTimestamps()
	default:
		return nil, ErrUnsupChunkType
	}
	// unreachable
}

func (c *Chunk) messageTimestamps() ([]int64, error) {
	ts := make([]int64, len(c.Messages))
	for i := range c.Messages {
		iTS, err := fasttime.TS2int(c.Messages[i].Timestamp)
		if err != nil {
			return nil, err
		}
		ts[i] = iTS
	}
	return ts, nil
}

// isInfo returns true, if the chunk is an info chunk.
func (g GroupID) isInfo() bool {
	return g[0] == catInfo
}

// isList returns true, if the chunk is a list chunk.
func (g GroupID) isList() bool {
	return g[0] == catList
}

// isSearch returns true, if the chunk is a search chunk.
func (g GroupID) isSearch() bool {
	return g[0] == catSearch
}

//go:generate mockgen -destination=mock_chunk/mock_chunk.go . Transformer

// Transformer is an interface that is called when the processor is finished
// processing a channel or thread.
type Transformer interface {
	// Transform is the function that starts the transformation of the channel
	// or thread with the given id.  It is called  when the reference count for
	// the channel id becomes zero (meaning, that there are no more chunks to
	// process).  It should return [transform.ErrClosed] if the transformer is
	// closed.
	Transform(ctx context.Context, channelID, threadID string, threadOnly bool) error
}

// NopTransformer is a transformer that does nothing.
type NopTransformer struct{}

func (*NopTransformer) Transform(context.Context, string, string, bool) error {
	return nil
}
