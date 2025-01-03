# Slackdump Chunk File Format

## Introduction

Chunk file format is a gzip-compressed JSONL file with each line being a JSON
object.

The benefit of chunk file format is that it can be converted to other formats,
such as Slack export format, or Slackdump format.  Chunk file format is used
internally by Slackdump during processing of the API output, it allows for
concurrent processing, minimising the memory usage during transformation
phase.  Each Chunk corresponds to a single API request that Slackdump issued
to Slack API.

## Chunk file format specification

The structure of the chunk file is better represented by the following code
snippet:

```go
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
	Count int `json:"n,omitempty"`

	// ThreadTS is populated if the chunk contains thread related data.  It
	// is Slack's thread_ts.
	ThreadTS string `json:"r,omitempty"`
	// IsLast is set to true if this is the last chunk for the channel or
	// thread.
	IsLast bool `json:"l,omitempty"`
	// NumThreads is the number of threads in the message chunk.
	NumThreads int `json:"nt,omitempty"`

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
```

Sample chunk JSON message:

```json
{
  "t": 5,
  "ts": 1683022288506765000,
  "id": "CHYLGDP0D",
  "ci": {
    "id": "CHYLGDP0D",
    "created": 1555493778,
    "is_open": false,
    "last_read": "1682743815.053209",
    "name_normalized": "random",
    "name": "random",
    //...
  }
}
```

## Fields

### t: Chunk type

Each JSON object can contain the following "chunk" of information, denoted as
unsigned 8-bit integer, each chunk type is a direct mapping to the Slack API
method that was used to retrieve the data:

- **Type 0**: slice of channel messages;
- **Type 1**: slice of channel message replies (a thread);
- **Type 2**: slice of files that were uploaded to the workspace (only definitions);
- **Type 3**: slice of users;
- **Type 4**: slice of channels;
- **Type 5**: channel information;
- **Type 6**: workspace information;
- **Type 7**: channel users;
- **Type 8**: starred items;
- **Type 9**: bookmarks;
- **Type 10**: search messages;
- **Type 11**: search files.

- **Type 0**: [conversations.history](https://api.slack.com/methods/conversations.history);
- **Type 1**: [conversations.replies](https://api.slack.com/methods/conversations.replies);
- **Type 2**: [files.list](https://api.slack.com/methods/files.list);
- **Type 3**: [users.list](https://api.slack.com/methods/users.list);
- **Type 4**: [conversations.list](https://api.slack.com/methods/conversations.list);
- **Type 5**: [conversations.info](https://api.slack.com/methods/conversations.info);
- **Type 6**: [auth.test](https://api.slack.com/methods/auth.test).
- **Type 7**: [conversation.members](https://api.slack.com/methods/conversations.members).
- **Type 8**: [stars.list](https://api.slack.com/methods/stars.list).
- **Type 9**: [bookmarks.list](https://api.slack.com/methods/bookmarks.list).
- **Type 10**: [search.messages](https://api.slack.com/methods/search.messages).
- **Type 11**: [search.files](https://api.slack.com/methods/search.files).

Message type value is guaranteed to be immutable in the future.  If a new
message type is added, it will be added as a new value, and the existing
values will not be changed.

### ts: Timestamp

The timestamp is a Unix timestamp in nanoseconds.  It contains the timestamp
of when the chunk was recorded.

### id: Channel ID

The channel ID is a string that contains the ID of the channel that the chunk
belongs to.  It is only populated for chunks of type 0, 1, and 2.

### n: Number of messages or files

The number of messages or files is an integer that contains the number of
messages or files that are contained in the chunk.  It is only populated for
relevant chunk types that return multiple records.

### r: Thread timestamp

The thread timestamp is a string that contains the timestamp of the thread
that the chunk belongs to.  It is only populated for chunks of type 1.

### l: Is last chunk

The is last chunk is a boolean that is set to true if the chunk is the last
chunk for the channel or thread.  It is only populated for chunks of type 0
and 1.

### nt: Number of threads

The number of threads is an integer that contains the number of threads that
are contained in the chunk.  It is only populated for chunks of type 0.

### ci: Channel information

The channel information contains the channel information as returned by the
API.

### cu: Channel users

The cu slice contains the user IDs of the users in the channel.  Those user IDs
should be looked up in the users slices, if they are present.

### p: Parent message

The parent message contains the parent message for a thread or a file chunk.

### m: Messages

The messages contains a chunk of messages.  It is only populated for chunks of
type 0 and 1.  This slice size can be in range from 1 to 1000 for message type
chunks.

### f: Files

The files contains a chunk of files.  It is only populated for chunks of type
2.

### u: Users

The users contains a chunk of users.  It is only populated for chunks of type
4.

### ch: Channels

The channels contains a chunk of channels.  It is only populated for chunks of
type 3.

### w: Workspace information

The workspace information contains the workspace information.  It is only
populated for chunks of type 5.

### st: Starred items

The starred items contains the starred items. (NOT IMPLEMENTED)

### b: Bookmarks

The bookmarks contains the bookmarks. (NOT IMPLEMENTED)

### sq: Search query

If this chunk is a search result, the search query is populated with the
search query that was used to retrieve the data.

### sm: Search messages

The search messages contains the search results.  It is a rather reduced version
of the slack.Message structure.

### sf: Search files

The search files contains the search results.  It is also, a reduced version
of the slack.File structure.
