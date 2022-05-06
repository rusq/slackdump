package main

import (
	"os"
	"path/filepath"
	"testing"
)

func Test_isExistingFile(t *testing.T) {
	testfile := filepath.Join(t.TempDir(), "cookies.txt")
	if err := os.WriteFile(testfile, []byte("blah"), 0600); err != nil {
		t.Fatal(err)
	}

	type args struct {
		cookie string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"not a file", args{"$blah"}, false},
		{"file", args{testfile}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isExistingFile(tt.args.cookie); got != tt.want {
				t.Errorf("isExistingFile() = %v, want %v", got, tt.want)
			}
		})
	}
}
