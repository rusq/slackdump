package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCacheDir(t *testing.T) {
	ucd, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		want string
	}{
		{
			"returns the cacheDir value",
			filepath.Join(ucd, cacheDirName),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CacheDir(); got != tt.want {
				t.Errorf("CacheDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ucd(t *testing.T) {
	type args struct {
		ucdFn func() (string, error)
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"detect OK",
			args{func() (string, error) { return "OK", nil }},
			filepath.Join("OK", cacheDirName),
		},
		{
			"detect failure",
			args{func() (string, error) { return "", errors.New("failed") }},
			".",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ucd(tt.args.ucdFn); got != tt.want {
				t.Errorf("ucd() = %v, want %v", got, tt.want)
			}
		})
	}
}
