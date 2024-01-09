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
type Chunk struct {
	Type          ChunkType               `json:"t"`
	Timestamp     int64                   `json:"ts"`
	ChannelID     string                  `json:"id,omitempty"`
	Count         int                     `json:"n,omitempty"`
	ThreadTS      string                  `json:"r,omitempty"`
	IsLast        bool                    `json:"l,omitempty"`
	NumThreads    int                     `json:"nt,omitempty"`
	Channel       *slack.Channel          `json:"ci,omitempty"`
	Parent        *slack.Message          `json:"p,omitempty"`
	Messages      []slack.Message         `json:"m,omitempty"`
	Files         []slack.File            `json:"f,omitempty"`
	Users         []slack.User            `json:"u,omitempty"`
	Channels      []slack.Channel         `json:"ch,omitempty"`
	WorkspaceInfo *slack.AuthTestResponse `json:"w,omitempty"`
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
unsigned 8-bit integer, each chunk type is a direct mapping to the Slack API method that was used to
retrieve the data:

- **Type 0**: slice of channel messages;
- **Type 1**: slice of channel message replies (a thread);
- **Type 2**: slice of files that were uploaded to the workspace (only definitions);
- **Type 3**: slice of channels;
- **Type 4**: slice of users;
- **Type 5**: workspace information.

- **Type 0**: [conversations.history](https://api.slack.com/methods/conversations.history);
- **Type 1**: [conversations.replies](https://api.slack.com/methods/conversations.replies);
- **Type 2**: [files.list](https://api.slack.com/methods/files.list);
- **Type 3**: [conversations.list](https://api.slack.com/methods/conversations.list);
- **Type 4**: [users.list](https://api.slack.com/methods/users.list);
- **Type 5**: [auth.test](https://api.slack.com/methods/auth.test).

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
chunks of type 0, 1, and 2.

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
API.  It is only populated for chunks of type 0, 1, and 2.

### p: Parent message

The parent message contains the parent message for a thread or a file chunk.
It is only populated for chunks of type 1 and 2.

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
