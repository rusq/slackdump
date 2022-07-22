package fixtures

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
)

// loadFixture loads a json data into T, or panics.
func Load[T any](js string) T {
	var ret T
	if err := json.Unmarshal([]byte(js), &ret); err != nil {
		panic(err)
	}
	return ret
}

// FilledBuffer returns buffer that filled with sz bytes of 0x00.
func FilledBuffer(sz int) *bytes.Buffer {
	var buf bytes.Buffer
	buf.Write(bytes.Repeat([]byte{0x00}, sz))
	return &buf
}

func FilledFile(sz int) *os.File {
	f, err := os.CreateTemp("", "sdunit*")
	if err != nil {
		panic(err)
	}
	f.Write(bytes.Repeat([]byte{0x00}, sz))
	f.Seek(0, io.SeekStart)
	return f
}
