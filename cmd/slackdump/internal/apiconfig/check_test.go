package apiconfig

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func Test_runConfigCheck(t *testing.T) {
	type args struct {
		args []string
	}
	tests := []struct {
		name    string
		args    args
		content []byte
		wantErr bool
	}{
		{
			"arg set, file exists, contents valid",
			args{args: []string{filepath.Join(t.TempDir(), "test.yml")}},
			[]byte(sampleLimitsYaml),
			false,
		},
		{
			"arg not set",
			args{},
			nil,
			true,
		},
		{
			"arg set, file not exists",
			args{args: []string{"not_here$$$.$$$"}},
			nil,
			true,
		},
		{
			"arg set, file exists, contents invalid",
			args{args: []string{filepath.Join(t.TempDir(), "test1.yml")}},
			[]byte("workers:-500"),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// if args and content is present, create this file.
			if len(tt.args.args) > 0 && len(tt.content) > 0 {
				if err := os.WriteFile(tt.args.args[0], tt.content, 0666); err != nil {
					t.Fatal(err)
				}
			}
			if err := runConfigCheck(context.Background(), CmdConfigCheck, tt.args.args); (err != nil) != tt.wantErr {
				t.Errorf("runConfigCheck() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
