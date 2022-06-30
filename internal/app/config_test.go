package app

import (
	"reflect"
	"testing"

	"github.com/rusq/slackdump/v2"
)

func TestConfig_compileValidateTemplate(t *testing.T) {
	type fields struct {
		ListFlags        ListFlags
		Input            Input
		Output           Output
		Oldest           TimeValue
		Latest           TimeValue
		FilenameTemplate string
		Options          slackdump.Options
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"id is ok",
			fields{FilenameTemplate: "{{.ID}}"},
			false,
		},
		{
			"name is ok",
			fields{FilenameTemplate: "{{.Name}}"},
			false,
		},
		{
			"just threadTS is not ok",
			fields{FilenameTemplate: "{{.ThreadTS}}"},
			true,
		},
		{
			"threadTS and message ID is ok",
			fields{FilenameTemplate: "{{.ID}}-{{.ThreadTS}}"},
			false,
		},
		{
			"threadTS and message ID is ok (conditional)",
			fields{FilenameTemplate: "{{.ID}}{{ if .ThreadTS}}-{{.ThreadTS}}{{end}}"},
			false,
		},
		{
			"message is not ok",
			fields{FilenameTemplate: "{{.Message}}"},
			true,
		},
		{
			"unknown field is not ok",
			fields{FilenameTemplate: "{{.Who_dis}}"},
			true,
		},
		{
			"empty not ok",
			fields{FilenameTemplate: ""},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Config{
				ListFlags:        tt.fields.ListFlags,
				Input:            tt.fields.Input,
				Output:           tt.fields.Output,
				Oldest:           tt.fields.Oldest,
				Latest:           tt.fields.Latest,
				FilenameTemplate: tt.fields.FilenameTemplate,
				Options:          tt.fields.Options,
			}
			if err := p.compileValidateTemplate(); (err != nil) != tt.wantErr {
				t.Errorf("Config.compileValidateTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func excl(s string) string {
	return string(excludeRune) + s
}

func TestInput_Load(t *testing.T) {
	type fields struct {
		List        []string
		ExcludeList []string
		Filename    string
	}
	type args struct {
		elements []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   Input
	}{
		{
			"ok",
			fields{},
			args{[]string{"1", "2", excl("3"), "4", excl("5")}},
			Input{
				List:        []string{"1", "2", "4"},
				ExcludeList: []string{"3", "5"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Input
			got.Load(tt.args.elements)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Load results mismatch: want=%+v, got=%+v", tt.want, got)
			}
		})
	}
}
