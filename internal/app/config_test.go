package app

import (
	"html/template"
	"testing"

	"github.com/rusq/slackdump"
)

func TestConfig_compileValidateTemplate(t *testing.T) {
	type fields struct {
		Creds            SlackCreds
		ListFlags        ListFlags
		Input            Input
		Output           Output
		Oldest           TimeValue
		Latest           TimeValue
		FilenameTemplate string
		tmpl             *template.Template
		Options          slackdump.Options
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"id is ok",
			fields{FilenameTemplate: "{{.ID}}", tmpl: template.New("")},
			false,
		},
		{
			"name is ok",
			fields{FilenameTemplate: "{{.Name}}", tmpl: template.New("")},
			false,
		},
		{
			"just threadTS is not ok",
			fields{FilenameTemplate: "{{.ThreadTS}}", tmpl: template.New("")},
			true,
		},
		{
			"threadTS and message ID is ok",
			fields{FilenameTemplate: "{{.ID}}-{{.ThreadTS}}", tmpl: template.New("")},
			false,
		},
		{
			"threadTS and message ID is ok (conditional)",
			fields{FilenameTemplate: "{{.ID}}{{ if .ThreadTS}}-{{.ThreadTS}}{{end}}", tmpl: template.New("")},
			false,
		},
		{
			"message is not ok",
			fields{FilenameTemplate: "{{.Message}}", tmpl: template.New("")},
			true,
		},
		{
			"unknown field is not ok",
			fields{FilenameTemplate: "{{.Who_dis}}", tmpl: template.New("")},
			true,
		},
		{
			"empty not ok",
			fields{FilenameTemplate: "", tmpl: template.New("")},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Config{
				Creds:            tt.fields.Creds,
				ListFlags:        tt.fields.ListFlags,
				Input:            tt.fields.Input,
				Output:           tt.fields.Output,
				Oldest:           tt.fields.Oldest,
				Latest:           tt.fields.Latest,
				FilenameTemplate: tt.fields.FilenameTemplate,
				tmpl:             tt.fields.tmpl,
				Options:          tt.fields.Options,
			}
			if err := p.compileValidateTemplate(); (err != nil) != tt.wantErr {
				t.Errorf("Config.compileValidateTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
