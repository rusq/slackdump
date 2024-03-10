// Package fileproc is the file processor that can be used in conjunction with
// the transformer.  It downloads files to the local filesystem using the
// provided downloader.  Probably it's a good idea to use the
// [downloader.Client] for this.
package fileproc

import (
	"testing"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/fixtures"
)

func TestIsValid(t *testing.T) {
	type args struct {
		f *slack.File
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"valid file",
			args{fixtures.LoadPtr[slack.File](fixtures.FileJPEG)},
			true,
		},
		{
			"tombstone",
			args{&slack.File{Mode: "tombstone", Name: "foo"}},
			false,
		},
		{
			"external file",
			args{&slack.File{Mode: "external", Name: "foo", IsExternal: true}},
			false,
		},
		{
			"hidden by limit",
			args{&slack.File{Mode: "hidden_by_limit", Name: "foo"}},
			false,
		},
		{
			"tombstone",
			args{&slack.File{Mode: "tombstone", Name: "foo"}},
			false,
		},
		{
			"external file",
			args{&slack.File{Mode: "", Name: "foo", IsExternal: true}},
			false,
		},
		{
			"empty name",
			args{&slack.File{Mode: "", Name: "", IsExternal: false}},
			false,
		},
		{
			"nil file",
			args{nil},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValid(tt.args.f); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}
