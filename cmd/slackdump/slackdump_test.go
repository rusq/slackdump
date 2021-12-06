package main

import "testing"

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
