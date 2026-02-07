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
