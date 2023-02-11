package fixtures

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"io"
)

// To generate the events.jsonl.gz file:
//   1. Record events from slackdump
//   2. Obfuscate them with ./cmd/slackdump diag obfuscate -i clear.jsonl -o events.jsonl
//   3. Compress them with gzip -9 -c events.jsonl > events.jsonl.gz

//go:embed assets/events.jsonl.gz
var eventsJsonlGz []byte

// EventsJSONL returns a reader for the events.jsonl.gz file.
func EventsJSONL() io.ReadCloser {
	gz, err := gzip.NewReader(bytes.NewReader(eventsJsonlGz))
	if err != nil {
		panic(err)
	}
	return gz
}
