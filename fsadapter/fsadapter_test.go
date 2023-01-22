package fsadapter

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tmp := t.TempDir()
	type args struct {
		name string
	}
	tests := []struct {
		name       string
		args       args
		wantString string
		wantErr    bool
	}{
		{
			"directory",
			args{filepath.Join(tmp, "blah")},
			"<directory: " + filepath.Join(tmp, "blah") + ">",
			false,
		},
		{
			"zip file",
			args{filepath.Join(tmp, "bloop.zip")},
			"<zip archive: " + filepath.Join(tmp, "bloop.zip") + ">",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("ForFilename() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			defer got.Close()

			assert.Equal(t, tt.wantString, fmt.Sprint(got))
		})
	}
}
