package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_loadSecrets(t *testing.T) {
	type args struct {
		files []string
	}
	tests := []struct {
		name    string
		args    args
		setupFn func(t *testing.T, dir string)
		wantEnv map[string]string
	}{
		{
			name: "loads secrets",
			args: args{
				files: []string{".env"},
			},
			setupFn: func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("DOT_ENV=set\n"), 0o666))
			},
			wantEnv: map[string]string{
				"DOT_ENV": "set",
			},
		},
		{
			"loads secrets from multiple files",
			args{
				files: []string{".env", "secrets.txt"},
			},
			func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("DOT_ENV=set\n"), 0o666))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "secrets.txt"), []byte("SECRETS_TXT=set\n"), 0o666))
			},
			map[string]string{
				"DOT_ENV":     "set",
				"SECRETS_TXT": "set",
			},
		},
		{
			"secrets from the second file don't override the first",
			args{
				files: []string{".env", "secrets.txt"},
			},
			func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("DOT_ENV=set\n"), 0o666))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "secrets.txt"), []byte("DOT_ENV=override\nSECRETS_TXT=set"), 0o666))
			},
			map[string]string{
				"DOT_ENV":     "set",
				"SECRETS_TXT": "set",
			},
		},
		{
			"does not override existing environment variables",
			args{
				files: []string{".env", "secrets.txt"},
			},
			func(t *testing.T, dir string) {
				t.Helper()
				t.Setenv("DOT_ENV", "env")
				require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("DOT_ENV=set\n"), 0o666))
			},
			map[string]string{
				"DOT_ENV": "env",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.setupFn != nil {
				tt.setupFn(t, dir)
			}
			for i := range tt.args.files {
				tt.args.files[i] = filepath.Join(dir, tt.args.files[i])
			}
			loadSecrets(tt.args.files)
			for k, v := range tt.wantEnv {
				require.Equal(t, v, os.Getenv(k))
			}
		})
	}
}

func Test_iftrue(t *testing.T) {
	type args[T any] struct {
		cond bool
		t    T
		f    T
	}
	tests := []struct {
		name string
		args args[int]
		want int
	}{
		{
			"returns true",
			args[int]{true, 1, 0},
			1,
		},
		{
			"returns false",
			args[int]{false, 1, 0},
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := iftrue(tt.args.cond, tt.args.t, tt.args.f); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("iftrue() = %v, want %v", got, tt.want)
			}
		})
	}
}
