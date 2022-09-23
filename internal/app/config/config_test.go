package config

import (
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
			p := &Params{
				ListFlags:        tt.fields.ListFlags,
				Input:            tt.fields.Input,
				Output:           tt.fields.Output,
				Oldest:           tt.fields.Oldest,
				Latest:           tt.fields.Latest,
				FilenameTemplate: tt.fields.FilenameTemplate,
				Options:          tt.fields.Options,
			}
			if err := p.compileValidateTemplate(); (err != nil) != tt.wantErr {
				t.Errorf("Params.compileValidateTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
