package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/internal/app/config"
	"github.com/rusq/slackdump/v2/internal/cache"
	"github.com/rusq/slackdump/v2/internal/structures"
)

func Test_output_validFormat(t *testing.T) {
	type fields struct {
		filename string
		format   string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{"empty", fields{}, false},
		{"empty", fields{format: config.OutputTypeJSON}, true},
		{"empty", fields{format: config.OutputTypeText}, true},
		{"empty", fields{format: "wtf"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := config.Output{
				Filename: tt.fields.filename,
				Format:   tt.fields.format,
			}
			if got := out.FormatValid(); got != tt.want {
				t.Errorf("Output.validFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_checkParameters(t *testing.T) {
	// test
	type args struct {
		args []string
	}
	tests := []struct {
		name    string
		args    args
		want    params
		wantErr bool
	}{
		{
			"channels",
			args{[]string{"-c", "-t", "x", "-cookie", "d"}},
			params{
				creds: cache.SlackCreds{
					Token:  "x",
					Cookie: "d",
				},
				appCfg: config.Params{
					ListFlags: config.ListFlags{
						Users:    false,
						Channels: true,
					},
					FilenameTemplate: defFilenameTemplate,
					Input:            config.Input{List: &structures.EntityList{}},
					Output:           config.Output{Filename: "-", Format: "text"},
					Limits:           slackdump.DefLimits,
				}},
			false,
		},
		{
			"users",
			args{[]string{"-u", "-t", "x", "-cookie", "d"}},
			params{
				creds: cache.SlackCreds{
					Token:  "x",
					Cookie: "d",
				},
				appCfg: config.Params{
					ListFlags: config.ListFlags{
						Channels: false,
						Users:    true,
					},
					FilenameTemplate: defFilenameTemplate,
					Input:            config.Input{List: &structures.EntityList{}},
					Output:           config.Output{Filename: "-", Format: "text"},
					Limits:           slackdump.DefLimits,
				}},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCmdLine(tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkParameters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_trunc(t *testing.T) {
	type args struct {
		s string
		n uint
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"empty", args{"", 7}, ""},
		{"few bytes", args{"abcdef", 2}, "ab"},
		{"zero", args{"abcdef", 0}, ""},
		{"same amount", args{"ab", 2}, "ab"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trunc(tt.args.s, tt.args.n); got != tt.want {
				t.Errorf("trunc() = %v, want %v", got, tt.want)
			}
		})
	}
}
