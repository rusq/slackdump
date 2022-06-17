package app

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rusq/slackdump/v2/auth"
)

func Test_isExistingFile(t *testing.T) {
	testfile := filepath.Join(t.TempDir(), "cookies.txt")
	if err := os.WriteFile(testfile, []byte("blah"), 0600); err != nil {
		t.Fatal(err)
	}

	type args struct {
		cookie string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"not a file", args{"$blah"}, false},
		{"file", args{testfile}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isExistingFile(tt.args.cookie); got != tt.want {
				t.Errorf("isExistingFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSlackCreds_Type(t *testing.T) {
	type fields struct {
		Token  string
		Cookie string
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    auth.Type
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := SlackCreds{
				Token:  tt.fields.Token,
				Cookie: tt.fields.Cookie,
			}
			got, err := c.Type(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("SlackCreds.Type() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SlackCreds.Type() = %v, want %v", got, tt.want)
			}
		})
	}
}
