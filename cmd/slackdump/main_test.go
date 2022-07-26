package main

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/internal/app"
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
		{"empty", fields{format: app.OutputTypeJSON}, true},
		{"empty", fields{format: app.OutputTypeText}, true},
		{"empty", fields{format: "wtf"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := app.Output{
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
	// setup
	slackdump.DefOptions.CacheDir = app.CacheDir()

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
				creds: app.SlackCreds{
					Token:  "x",
					Cookie: "d",
				},
				appCfg: app.Config{
					ListFlags: app.ListFlags{
						Users:    false,
						Channels: true,
					},
					FilenameTemplate: defFilenameTemplate,
					Input:            app.Input{List: &structures.EntityList{}},
					Output:           app.Output{Filename: "-", Format: "text"},
					Options:          slackdump.DefOptions,
				}},
			false,
		},
		{
			"users",
			args{[]string{"-u", "-t", "x", "-cookie", "d"}},
			params{
				creds: app.SlackCreds{
					Token:  "x",
					Cookie: "d",
				},
				appCfg: app.Config{
					ListFlags: app.ListFlags{
						Channels: false,
						Users:    true,
					},
					FilenameTemplate: defFilenameTemplate,
					Input:            app.Input{List: &structures.EntityList{}},
					Output:           app.Output{Filename: "-", Format: "text"},
					Options:          slackdump.DefOptions,
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

func Test_banner(t *testing.T) {
	tests := []struct {
		name  string
		wantW string
	}{
		{
			"make sure I haven't fucked up",
			fmt.Sprintf(bannerFmt, build, buildYear, trunc(commit, 7)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			banner(w)
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("banner() = %v, want %v", gotW, tt.wantW)
			}
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
