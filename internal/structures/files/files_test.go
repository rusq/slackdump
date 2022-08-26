package files

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/internal/fixtures"
)

func Test_addToken(t *testing.T) {
	type args struct {
		uri   string
		token string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"ok",
			args{"https://slack.com/files/BLAHBLAH/x.jpg", "xoxe-xxxxx"},
			"https://slack.com/files/BLAHBLAH/x.jpg?t=xoxe-xxxxx",
			false,
		},
		{
			"replace existing",
			args{"https://slack.com/files/BLAHBLAH/x.jpg?t=xoxe-yyyyy", "xoxe-xxxxx"},
			"https://slack.com/files/BLAHBLAH/x.jpg?t=xoxe-xxxxx",
			false,
		},
		{
			"preseves other parameters",
			args{"https://slack.com/files/BLAHBLAH/x.jpg?t=xoxe-yyyyy&q=bbbb", "xoxe-xxxxx"},
			"https://slack.com/files/BLAHBLAH/x.jpg?q=bbbb&t=xoxe-xxxxx",
			false,
		},
		{
			"fails on invalid URL",
			args{"://test", "x"},
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := addToken(tt.args.uri, tt.args.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("addToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("addToken() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateTokenFn(t *testing.T) {
	const testToken = "xoxe-1234"
	fn := UpdateTokenFn(testToken)
	t.Run("adds the token to url and thumbnail fields", func(t *testing.T) {
		file := fixtures.Load[slack.File](fixtures.FileJPEG)
		if err := fn(&file); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		wantfile := fixtures.Load[slack.File](strings.ReplaceAll(fileTokenPlaceholder, "%TOKEN%", testToken))
		if !reflect.DeepEqual(file, wantfile) {
			t.Errorf("files are different")
		}
	})
	t.Run("fails on invalid URL", func(t *testing.T) {
		file := slack.File{URLPrivateDownload: "://what is this?"}
		if err := fn(&file); err == nil {
			t.Errorf("expected an error, but got nil")
		}
	})
	t.Run("returns on empty token with nil", func(t *testing.T) {
		emptyTokenFn := UpdateTokenFn("")
		file := slack.File{URLPrivateDownload: "://what is this?"}
		if err := emptyTokenFn(&file); err != nil {
			t.Errorf("unexpected  error: %s", err)
		}
	})
}

func TestUpdatePathFn(t *testing.T) {
	const testpath = "/testpath"
	fn := UpdatePathFn(testpath)
	t.Run("ensure path is updated", func(t *testing.T) {
		file := fixtures.Load[slack.File](fixtures.FileJPEG)
		if err := fn(&file); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		wantfile := fixtures.Load[slack.File](fixtures.FileJPEG)
		wantfile.URLPrivate = testpath
		wantfile.URLPrivateDownload = testpath
		if !reflect.DeepEqual(file, wantfile) {
			t.Errorf("files are different")
		}
	})
}

func Test_callForEach(t *testing.T) {
	t.Run("iterates properly", func(t *testing.T) {
		var testSlice = []int{1, 2, 3}
		var got = []*int{&testSlice[0], &testSlice[1], &testSlice[2]}
		err := callForEach(got, func(el *int) error {
			*el = *el * (*el)
			return nil
		})
		if err != nil {
			t.Fatalf("unexpected error")
		}
		want := []int{1, 4, 9}
		if !reflect.DeepEqual(want, testSlice) {
			t.Errorf("callForEach(): want=%v, got=%v", want, got)
		}
	})
	t.Run("propagates error", func(t *testing.T) {
		var testSlice = []int{1, 2, 3}
		var got = []*int{&testSlice[0], &testSlice[1], &testSlice[2]}
		err := callForEach(got, func(el *int) error {
			return errors.New("not today, sir")
		})
		if err == nil {
			t.Errorf("expected an error, but got nil")
		}
	})
}

const fileTokenPlaceholder = `{
	"id": "F02PM6A1AUA",
	"created": 1638784624,
	"timestamp": 1638784624,
	"name": "Chevy.jpg",
	"title": "Chevy.jpg",
	"mimetype": "image/jpeg",
	"image_exif_rotation": 0,
	"filetype": "jpg",
	"pretty_type": "JPEG",
	"user": "UHSD97ZA5",
	"mode": "hosted",
	"editable": false,
	"is_external": false,
	"external_type": "",
	"size": 359002,
	"url": "",
	"url_download": "",
	"url_private": "https://files.slack.com/files-pri/THY5HTZ8U-F02PM6A1AUA/chevy.jpg?t=%TOKEN%",
	"url_private_download": "https://files.slack.com/files-pri/THY5HTZ8U-F02PM6A1AUA/download/chevy.jpg?t=%TOKEN%",
	"original_h": 1080,
	"original_w": 1920,
	"thumb_64": "https://files.slack.com/files-tmb/THY5HTZ8U-F02PM6A1AUA-d1fdb12d3e/chevy_64.jpg?t=%TOKEN%",
	"thumb_80": "https://files.slack.com/files-tmb/THY5HTZ8U-F02PM6A1AUA-d1fdb12d3e/chevy_80.jpg?t=%TOKEN%",
	"thumb_160": "https://files.slack.com/files-tmb/THY5HTZ8U-F02PM6A1AUA-d1fdb12d3e/chevy_160.jpg?t=%TOKEN%",
	"thumb_360": "https://files.slack.com/files-tmb/THY5HTZ8U-F02PM6A1AUA-d1fdb12d3e/chevy_360.jpg?t=%TOKEN%",
	"thumb_360_gif": "",
	"thumb_360_w": 360,
	"thumb_360_h": 203,
	"thumb_480": "https://files.slack.com/files-tmb/THY5HTZ8U-F02PM6A1AUA-d1fdb12d3e/chevy_480.jpg?t=%TOKEN%",
	"thumb_480_w": 480,
	"thumb_480_h": 270,
	"thumb_720": "https://files.slack.com/files-tmb/THY5HTZ8U-F02PM6A1AUA-d1fdb12d3e/chevy_720.jpg?t=%TOKEN%",
	"thumb_720_w": 720,
	"thumb_720_h": 405,
	"thumb_960": "https://files.slack.com/files-tmb/THY5HTZ8U-F02PM6A1AUA-d1fdb12d3e/chevy_960.jpg?t=%TOKEN%",
	"thumb_960_w": 960,
	"thumb_960_h": 540,
	"thumb_1024": "https://files.slack.com/files-tmb/THY5HTZ8U-F02PM6A1AUA-d1fdb12d3e/chevy_1024.jpg?t=%TOKEN%",
	"thumb_1024_w": 1024,
	"thumb_1024_h": 576,
	"permalink": "https://ora600.slack.com/files/UHSD97ZA5/F02PM6A1AUA/chevy.jpg",
	"permalink_public": "https://slack-files.com/THY5HTZ8U-F02PM6A1AUA-ea648a3dee",
	"edit_link": "",
	"preview": "",
	"preview_highlight": "",
	"lines": 0,
	"lines_more": 0,
	"is_public": true,
	"public_url_shared": false,
	"channels": null,
	"groups": null,
	"ims": null,
	"initial_comment": {},
	"comments_count": 0,
	"num_stars": 0,
	"is_starred": false,
	"shares": {
	  "public": null,
	  "private": null
	}
  }`
