// This package is based on the Golang source code with some modifications.
//
// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package help

import (
	"errors"
	"testing"
	"text/template"
)

func Test_RenderTemplate(t *testing.T) {
	type args struct {
		tmpl string
		data any
	}
	tests := []struct {
		name        string
		args        args
		want        string
		wantErr     bool
		wantExecErr bool
	}{
		{
			name:    "all ok",
			args:    args{"unit test: {{ .name }}", map[string]string{"name": "Name"}},
			want:    "unit test: Name",
			wantErr: false,
		},
		{
			name:        "variable not defined",
			args:        args{"test: {{ .undefined }}", map[string]string{"other": "value"}},
			want:        "",
			wantErr:     true,
			wantExecErr: true,
		},
		{
			name:        "missing key in conditional",
			args:        args{"{{ if .undefined }}present{{ end }}", map[string]string{"other": "value"}},
			wantErr:     true,
			wantExecErr: true,
		},
		{
			name: "present but empty value",
			args: args{"before[{{ .name }}]after", map[string]string{"name": ""}},
			want: "before[]after",
		},
		{
			name:    "malformed template",
			args:    args{"before {{ .name", map[string]string{"name": "Name"}},
			wantErr: true,
		},
		{
			name: "value contains placeholder text",
			args: args{"test: {{ .name }}", map[string]string{"name": "<no value>"}},
			want: "test: <no value>",
		},
		{
			name:    "renders structs too",
			args:    args{"this is a field on a struct: {{ .Field1 }}", struct{ Field1 int }{Field1: 42}},
			want:    "this is a field on a struct: 42",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderTemplate(tt.args.tmpl, tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RenderTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}
			var execErr template.ExecError
			if errors.As(err, &execErr) != tt.wantExecErr {
				t.Fatalf("RenderTemplate() error = %v, want template.ExecError: %v", err, tt.wantExecErr)
			}
			if got != tt.want {
				t.Errorf("RenderTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}
