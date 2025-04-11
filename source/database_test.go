package source

import (
	"testing"

	"github.com/rusq/slackdump/v3/internal/chunk/backend/dbase"
)

func TestDatabase_Name(t *testing.T) {
	type fields struct {
		name    string
		files   Storage
		avatars Storage
		Source  *dbase.Source
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "test",
			fields: fields{
				name:    "foobar",
				files:   NoStorage{},
				avatars: NoStorage{},
			},
			want: "foobar",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Database{
				name:    tt.fields.name,
				files:   tt.fields.files,
				avatars: tt.fields.avatars,
				Source:  tt.fields.Source,
			}
			if got := d.Name(); got != tt.want {
				t.Errorf("Database.Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDatabase_Type(t *testing.T) {
	type fields struct {
		name    string
		files   Storage
		avatars Storage
		Source  *dbase.Source
	}
	tests := []struct {
		name   string
		fields fields
		want   Flags
	}{
		{
			name: "test",
			want: FDatabase,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Database{
				name:    tt.fields.name,
				files:   tt.fields.files,
				avatars: tt.fields.avatars,
				Source:  tt.fields.Source,
			}
			if got := d.Type(); got != tt.want {
				t.Errorf("Database.Type() = %v, want %v", got, tt.want)
			}
		})
	}
}
