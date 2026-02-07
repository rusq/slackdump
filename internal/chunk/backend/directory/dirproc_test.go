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
