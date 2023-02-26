package fixtures

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"io"
)

// To generate the chunks.jsonl.gz file:
//
//  ./slackdump diag record stream <channel> | ./slackdump diag obfuscate | gzip -9 -c > chunks.jsonl.gz

//go:embed assets/chunks.jsonl.gz
var chunksJsonlGz []byte

// ChunksJSONL returns a reader for the chunks.jsonl.gz file.  Reader must be
// closed
func ChunksJSONL() io.ReadCloser {
	gz, err := gzip.NewReader(bytes.NewReader(chunksJsonlGz))
	if err != nil {
		panic(err)
	}
	return gz
}
