// Package directory is a processor that writes the data into gzipped files in a
// directory.  Each conversation is output to a separate gzipped JSONL file.
// If a thread is given, the filename will have the thread ID in it.
package directory

import (
	"sync/atomic"
	"testing"

	"github.com/rusq/slackdump/v3/internal/chunk"
)

type mockWriteCloser struct {
	WriteCalled atomic.Bool
	CloseCalled atomic.Bool
}

func (m *mockWriteCloser) Write(p []byte) (n int, err error) {
	m.WriteCalled.Store(true)
	return 0, nil
}

func (m *mockWriteCloser) Close() error {
	m.CloseCalled.Store(true)
	return nil
}

func Test_dirproc_Close(t *testing.T) {
	tests := []struct {
		name    string
		fields  *dirproc
		prep    func(d *dirproc)
		wantErr bool
	}{
		{
			"already closed",
			&dirproc{},
			func(d *dirproc) {
				d.closed.Store(true)
			},
			false,
		},
		{
			"close ok",
			&dirproc{
				Recorder: &chunk.Recorder{},
				wc:       &mockWriteCloser{},
			},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prep != nil {
				tt.prep(tt.fields)
			}
			if err := tt.fields.Close(); (err != nil) != tt.wantErr {
				t.Errorf("directory.Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
