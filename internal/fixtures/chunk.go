package fixtures

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"io"
)

// To generate the chunks.jsonl.gz file:
//
//  ./slackdump tools record stream <channel> | ./slackdump tools obfuscate | gzip -9 -c > chunks.jsonl.gz

//go:embed assets/chunks.jsonl.gz
var chunksJsonlGz []byte

const ChunkFileChannelID = "CO73D19AAE17"

// chunksJSONL returns a reader for the b []byte, which assumed to be a
// gzip-compressed bytes slice. It panics on error.
func chunksJSONL(b []byte) io.ReadSeeker {
	gz, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		panic(err)
	}
	defer gz.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	if err != nil {
		panic(err)
	}

	return bytes.NewReader(buf.Bytes())
}

// ChunkFileJSONL returns a reader for the chunks.jsonl.gz file.
func ChunkFileJSONL() io.ReadSeeker {
	return chunksJSONL(chunksJsonlGz)
}

const (
	ChunkWorkspace = `{"t":6,"ts":1682508612986310000,"w":{"url":"https://unittest.slack.com/","team":"Unittest Team","user":"charlie","team_id":"T02A8XRGA","user_id":"US5ATREAF","bot_id":""}}`
)
