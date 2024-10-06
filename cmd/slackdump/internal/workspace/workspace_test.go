package workspace

import (
	"os"
	"path/filepath"
	"testing"

	fx "github.com/rusq/slackdump/v3/internal/fixtures"
)

func Test_argsWorkspace(t *testing.T) {
	type args struct {
		args       []string
		defaultWsp string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"empty",
			args{[]string{}, ""},
			"",
		},
		{
			"default is set, no workspace in args",
			args{[]string{}, "default"},
			"default",
		},
		{
			"default overrides args args",
			args{[]string{"arg"}, "default"},
			"default",
		},
		{
			"returns must be lowercase",
			args{[]string{"UPPERCASE"}, "DEFAULT"},
			"default",
		},
		{
			"returns must be lowercase",
			args{[]string{"UPPERCASE"}, ""},
			"uppercase",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := argsWorkspace(tt.args.args, tt.args.defaultWsp); got != tt.want {
				t.Errorf("argsWorkspace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCurrent(t *testing.T) {
	//fixtures

	empty := t.TempDir()

	// case1 has files, but no file pointing to the current workspace
	case1 := t.TempDir()
	fx.PrepareDir(t, case1, "dummy", fx.WorkspaceFiles...)

	// case2 has files, and a pointer to the current workspace.
	case2 := t.TempDir()
	fx.PrepareDir(t, case2, "dummy", fx.WorkspaceFiles...)
	os.WriteFile(filepath.Join(case2, "workspace.txt"), []byte(fx.StripExt(fx.WorkspaceFiles[0])+"\n"), 0644)

	// case3 has a file, which is specified as a directory to the function
	// so that manager fails to initialise.
	case3 := t.TempDir()
	os.WriteFile(filepath.Join(case3, "cache_dir"), []byte(""), 0644)

	// case4 workspace pointer points to non-existing file.
	case4 := t.TempDir()
	fx.PrepareDir(t, case4, "dummy", fx.WorkspaceFiles...)
	os.WriteFile(filepath.Join(case4, "workspace.txt"), []byte("doesnotexist\n"), 0644)

	// tests
	type args struct {
		cacheDir string
		override string
	}
	tests := []struct {
		name    string
		args    args
		wantWsp string
		wantErr bool
	}{
		{
			"empty,no override",
			args{empty, ""},
			"default",
			false,
		},
		{
			"override, does not exist",
			args{empty, "override"},
			"",
			true,
		},
		{
			"case1, no override",
			args{case1, ""},
			"default",
			false,
		},
		{
			"case2, no override",
			args{case2, ""},
			fx.StripExt(fx.WorkspaceFiles[0]),
			false,
		},
		{
			"case2, override",
			args{case2, fx.StripExt(fx.WorkspaceFiles[1])},
			fx.StripExt(fx.WorkspaceFiles[1]),
			false,
		},
		{
			"case2, override, does not exist",
			args{case2, "doesnotexist"},
			"",
			true,
		},
		{
			"invalid directory",
			args{filepath.Join(case3, "cache_dir"), ""},
			"",
			true,
		},
		{
			"case4, no override, returns default",
			args{case4, ""},
			"default",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWsp, err := Current(tt.args.cacheDir, tt.args.override)
			if (err != nil) != tt.wantErr {
				t.Errorf("Current() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotWsp != tt.wantWsp {
				t.Errorf("Current() = %v, want %v", gotWsp, tt.wantWsp)
			}
		})
	}
}
