package cache

import (
	"errors"
	"io"
	"sort"
	"strings"
	"testing"

	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/stretchr/testify/assert"
)

func Test_currentWsp(t *testing.T) {
	type args struct {
		r io.Reader
	}
	tests := []struct {
		name string
		m    *Manager
		args args
		want string
	}{
		{
			"ok",
			&Manager{dir: "test"},
			args{strings.NewReader("foo\n")},
			"foo",
		},
		{
			"empty",
			&Manager{dir: "test"},
			args{strings.NewReader("")},
			defCredsFile,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.readWsp(tt.args.r); got != tt.want {
				t.Errorf("currentWsp() = %v, want %v", got, tt.want)
			}
		})
	}
}

func prepareDir(t *testing.T, dir string) {
	fixtures.PrepareDir(t, dir, "dummy", fixtures.WorkspaceFiles...)
}

func testFiles(dir string) []string {
	return fixtures.JoinPath(dir, fixtures.WorkspaceFiles...)
}

func TestManager_listFiles(t *testing.T) {
	tests := []struct {
		name    string
		prepFn  func(t *testing.T, dir string)
		want    func(dir string) []string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			"ensure that it returns a list of files",
			func(t *testing.T, dir string) {
				prepareDir(t, dir)
			},
			func(dir string) []string {
				return testFiles(dir)
			},
			func(t assert.TestingT, err error, i ...interface{}) bool {
				return err == nil
			},
		},
		{
			"empty",
			func(t *testing.T, dir string) {},
			nil,
			func(t assert.TestingT, err error, i ...interface{}) bool {
				return errors.Is(err, ErrNoWorkspaces)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempdir := t.TempDir()
			m := &Manager{
				dir: tempdir,
			}
			if tt.prepFn != nil {
				tt.prepFn(t, tempdir)
			}
			got, err := m.listFiles()
			if !tt.wantErr(t, err, "List()") {
				return
			}
			var want []string
			if tt.want != nil {
				want = tt.want(tempdir)
			}
			sort.Strings(want)
			assert.Equalf(t, want, got, "List()")
		})
	}
}

func TestManager_ExistsErr(t *testing.T) {
	t.Parallel()
	t.Run("empty directory", func(t *testing.T) {
		t.Parallel()

		tempdir := t.TempDir()
		m := &Manager{
			dir: tempdir,
		}
		err := m.ExistsErr("foo")
		assert.ErrorIs(t, err, ErrNoWorkspaces)
	})
	t.Run("workspace exists", func(t *testing.T) {
		t.Parallel()

		tempdir := t.TempDir()
		prepareDir(t, tempdir)
		m := &Manager{
			dir: tempdir,
		}
		err := m.ExistsErr("foo")
		assert.NoError(t, err)
	})
	t.Run("workspace does not exist", func(t *testing.T) {
		t.Parallel()

		tempdir := t.TempDir()
		prepareDir(t, tempdir)
		m := &Manager{
			dir: tempdir,
		}
		err := m.ExistsErr("baz")
		var e *ErrWorkspace
		assert.ErrorAs(t, err, &e)
		assert.Equal(t, e.Message, "no such workspace")
		assert.Equal(t, e.Workspace, "baz")
	})
}
