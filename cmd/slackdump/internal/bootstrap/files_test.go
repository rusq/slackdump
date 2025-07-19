package bootstrap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
)

func sayBool(b bool) func(t *testing.T) string {
	return func(t *testing.T) string {
		t.Helper()
		oldYesNo := yesno
		t.Cleanup(func() {
			yesno = oldYesNo
		})
		yesno = func(_ string) bool {
			return b
		}
		return ""
	}
}

var sayYes = sayBool(true)
var sayNo = sayBool(false)

func TestAskOverwrite(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name         string
		args         args
		GlobalYesMan bool
		setup        func(t *testing.T) string //should return temp directory path.
		wantErr      bool
	}{
		{
			name:         "YesMan set to true",
			args:         args{"somefile.txt"},
			GlobalYesMan: true,
			setup:        sayNo, // should be ignored
			wantErr:      false,
		},
		{
			name:         "path does not exist",
			args:         args{"i_do_not_exist.txt"},
			GlobalYesMan: false,
			setup:        sayNo, // should be ignored, because file does not exist
			wantErr:      false,
		},
		{
			name:         "file exists, and we say NO",
			args:         args{"i_exist.txt"},
			GlobalYesMan: false,
			setup: func(t *testing.T) string {
				sayNo(t)
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "i_exist.txt"), []byte("i think therefore i exist"), 0644); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			wantErr: true,
		},
		{
			name:         "file exists, and we say YES",
			args:         args{"i_exist.txt"},
			GlobalYesMan: false,
			setup: func(t *testing.T) string {
				sayYes(t)
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "i_exist.txt"), []byte("i think therefore i exist"), 0644); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			wantErr: false,
		},
		{
			name:         "dir exists, and we say YES",
			args:         args{"dir_exists"},
			GlobalYesMan: false,
			setup: func(t *testing.T) string {
				sayYes(t)
				dir := t.TempDir()
				if err := os.Mkdir(filepath.Join(dir, "dir_exists"), 0755); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			wantErr: false,
		},
		{
			name:         "dir exists, and we say NO",
			args:         args{"dir_exists2"},
			GlobalYesMan: false,
			setup: func(t *testing.T) string {
				sayNo(t)
				dir := t.TempDir()
				if err := os.Mkdir(filepath.Join(dir, "dir_exists2"), 0755); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg.YesMan = tt.GlobalYesMan
			dir := tt.setup(t)
			if err := AskOverwrite(filepath.Join(dir, tt.args.path)); (err != nil) != tt.wantErr {
				t.Errorf("AskOverwrite() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
