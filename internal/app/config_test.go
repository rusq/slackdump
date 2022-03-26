package app

import (
	"os"
	"path/filepath"
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
				Creds:            tt.fields.Creds,
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

func TestSlackCreds_Validate(t *testing.T) {
	// prepare environment
	tempdir := t.TempDir()

	notafile := filepath.Join(tempdir, "fakecookie.txt")
	if err := os.Mkdir(notafile, 0700); err != nil {
		t.Fatal(err)
	}

	testfile := filepath.Join(tempdir, "cookie.txt")
	if err := os.WriteFile(testfile, []byte("xxx"), 0600); err != nil {
		t.Fatal(err)
	}
	// end prep

	type fields struct {
		Token  string
		Cookie string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"empty",
			fields{},
			true,
		},
		{
			"token missing",
			fields{Cookie: "$hey"},
			true,
		},
		{
			"cookie missing",
			fields{Token: "$hey"},
			true,
		},
		{
			"all ok, cookie is not a file",
			fields{Token: "$tok", Cookie: "$hey"},
			false,
		},
		{
			"all ok, cookie is a file",
			fields{Token: "$tok", Cookie: testfile},
			false,
		},
		{
			"all ok, cookie is a fs object, but not a file",
			fields{Token: "$tok", Cookie: notafile},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := SlackCreds{
				Token:  tt.fields.Token,
				Cookie: tt.fields.Cookie,
			}
			if err := c.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("SlackCreds.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
