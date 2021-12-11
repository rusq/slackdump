package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
		{"empty", fields{format: outputTypeJSON}, true},
		{"empty", fields{format: outputTypeText}, true},
		{"empty", fields{format: "wtf"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := output{
				filename: tt.fields.filename,
				format:   tt.fields.format,
			}
			if got := out.validFormat(); got != tt.want {
				t.Errorf("output.validFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_checkParameters(t *testing.T) {
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
				list: listFlags{
					users:    false,
					channels: true,
				},
				creds: slackCreds{
					token:  "x",
					cookie: "d",
				},
				output:           output{filename: "-", format: "text"},
				channelsToExport: []string{},
			},
			false,
		},
		{
			"users",
			args{[]string{"-u", "-t", "x", "-cookie", "d"}},
			params{
				list: listFlags{
					channels: false,
					users:    true,
				},
				creds: slackCreds{
					token:  "x",
					cookie: "d",
				},
				output:           output{filename: "-", format: "text"},
				channelsToExport: []string{},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkParameters(tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkParameters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
