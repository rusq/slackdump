// Package pwcompat provides a compatibility layer, so when the playwright-go
// team decides to break compatibility again, there's a place to write a
// workaround.
//
//go:build: darwin
package pwcompat

import (
	"os"
	"path/filepath"
	"testing"
)

func bailonerr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func Test_getDefaultCacheDirectory(t *testing.T) {
	t.Parallel()
	home, err := os.UserHomeDir()
	bailonerr(t, err)
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		{
			"darwin",
			filepath.Join(home, "Library", "Caches"),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := getDefaultCacheDirectory()
			if (err != nil) != tt.wantErr {
				t.Errorf("getDefaultCacheDirectory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getDefaultCacheDirectory() = %v, want %v", got, tt.want)
			}
		})
	}
}
