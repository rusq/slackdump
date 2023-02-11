package fixtures

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"io"
)

// To generate the events.jsonl.gz file:
//
//  ./slackdump diag record <channel> | ./slackdump diag obfuscate | gzip -9 -c > events.jsonl.gz

//go:embed assets/events.jsonl.gz
var eventsJsonlGz []byte

// EventsJSONL returns a reader for the events.jsonl.gz file.  Reader must be
// closed
func EventsJSONL() io.ReadCloser {
	gz, err := gzip.NewReader(bytes.NewReader(eventsJsonlGz))
	if err != nil {
		panic(err)
	}
	return gz
}
