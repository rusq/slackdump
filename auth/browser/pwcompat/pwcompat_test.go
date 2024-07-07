// Package pwcompat provides a compatibility layer, so when the playwright-go
// team decides to break compatibility again, there's a place to write a
// workaround.
package pwcompat

import (
	"testing"

	"github.com/playwright-community/playwright-go"
)

func TestNewDriver(t *testing.T) {
	t.Parallel()
	type args struct {
		runopts *playwright.RunOptions
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"default dir",
			args{&playwright.RunOptions{
				DriverDirectory:     "",
				SkipInstallBrowsers: true,
				Browsers:            []string{"chrome"}},
			},
			false,
		},
		{
			"custom dir",
			args{&playwright.RunOptions{
				DriverDirectory:     t.TempDir(),
				SkipInstallBrowsers: true,
				Browsers:            []string{"chrome"}},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewDriver(tt.args.runopts)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDriver() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
