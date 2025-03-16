package chunk

import (
	"errors"
	"io"
	"testing"
)

func TestPlayer_Thread(t *testing.T) {
	rs, err := FromReader(marshalChunks(testThreads...))
	if err != nil {
		t.Fatal(err)
	}
	p := Player{
		f:       rs,
		pointer: make(offsets),
	}
	m, err := p.Thread("C1234567890", "1234567890.123456")
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(m))
	}
	// again
	m, err = p.Thread("C1234567890", "1234567890.123456")
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(m))
	}
	// should error
	m, err = p.Thread("C1234567890", "1234567890.123456")
	if !errors.Is(err, io.EOF) {
		t.Error(err, "expected io.EOF")
	}
	if len(m) > 0 {
		t.Fatalf("expected 0 messages, got %d", len(m))
	}
}
