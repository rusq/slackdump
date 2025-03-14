package repository

import (
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
)

func TestNewDBSearchFile(t *testing.T) {
	type args struct {
		chunkID int64
		n       int
		sf      *slack.File
	}
	tests := []struct {
		name    string
		args    args
		want    *DBSearchFile
		wantErr bool
	}{
		{
			name: "creates a new DBSearchFile",
			args: args{chunkID: 42, n: 50, sf: file1},
			want: &DBSearchFile{
				ID:      0, // autoincrement, handled by the database.
				ChunkID: 42,
				FileID:  "FILE1",
				Index:   50,
				Data:    must(marshal(file1)),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDBSearchFile(tt.args.chunkID, tt.args.n, tt.args.sf)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDBSearchFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
