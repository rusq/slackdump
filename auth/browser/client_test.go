package browser

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
)

func Test_float2time(t *testing.T) {
	type args struct {
		v float64
	}
	tests := []struct {
		name string
		args args
		want time.Time
	}{
		{"ok", args{1.68335956e+09}, time.Unix(1683359560, 0)},
		{"stripped", args{1.6544155598311e+09}, time.Unix(1654415559, 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := float2time(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("float2time() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_pwRepair(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on windows")
	}
	t.Run("known executable permissions problem causes reinstall", func(t *testing.T) {
		baseDir := t.TempDir()
		fakePwDir := filepath.Join(baseDir, "playwright-99.20.0")

		// installCalledi should be set to true if the install function is
		// called.
		installCalled := false
		// set the mock install functions.
		oldInstall := installFn
		defer func() { installFn = oldInstall }()
		installFn = func(...*playwright.RunOptions) error {
			installCalled = true
			return nil
		}
		oldNewDriverFn := newDriverFn
		defer func() { newDriverFn = oldNewDriverFn }()
		newDriverFn = func(*playwright.RunOptions) (*playwright.PlaywrightDriver, error) {
			return &playwright.PlaywrightDriver{
				DriverDirectory: fakePwDir,
			}, nil
		}

		// create a fake node file with the wrong permissions.
		makeFakeNode(t, fakePwDir, 0o644)
		// run the repair function.
		runopts := &playwright.RunOptions{
			Browsers: []string{"chromium"},
		}
		if err := pwRepair(runopts); err != nil {
			t.Fatal(err)
		}

		if !installCalled {
			t.Fatal("install was not called")
		}
		// check that the directory was removed
		if _, err := os.Stat(fakePwDir); !os.IsNotExist(err) {
			t.Fatal("directory was not removed")
		}
	})
}

func makeFakeNode(t *testing.T, dir string, mode fs.FileMode) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "node"), []byte("hello"), mode); err != nil {
		t.Fatal(err)
	}
}

func Test_pwIsKnownProblem(t *testing.T) {
	t.Run("known executable permissions problem", func(t *testing.T) {
		baseDir := t.TempDir()
		makeFakeNode(t, baseDir, 0o644)
		if err := pwIsKnownProblem(baseDir); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("other problem", func(t *testing.T) {
		baseDir := t.TempDir()
		makeFakeNode(t, baseDir, 0o755)
		err := pwIsKnownProblem(baseDir)
		if err == nil {
			t.Fatal("unexpected success")
		}
		if !errors.Is(err, errUnknownProblem) {
			t.Fatal("unexpected error")
		}
	})
}
