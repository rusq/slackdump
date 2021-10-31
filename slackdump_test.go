package slackdump

import (
	"reflect"
	"testing"
	"time"
)

func Test_maxStringLength(t *testing.T) {
	type args struct {
		strings []string
	}
	tests := []struct {
		name       string
		args       args
		wantMaxlen int
	}{
		{"ascii", args{[]string{"123", "abc", "defg"}}, 4},
		{"unicode", args{[]string{"сообщение1", "проверка", "тест"}}, 10},
		{"empty", args{[]string{}}, 0},
		{"several empty", args{[]string{"", "", "", ""}}, 0},
		{"several empty one full", args{[]string{"", "", "1", ""}}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotMaxlen := maxStringLength(tt.args.strings); gotMaxlen != tt.wantMaxlen {
				t.Errorf("maxStringLength() = %v, want %v", gotMaxlen, tt.wantMaxlen)
			}
		})
	}
}

func Test_fromSlackTime(t *testing.T) {
	type args struct {
		timestamp string
	}
	tests := []struct {
		name    string
		args    args
		want    time.Time
		wantErr bool
	}{
		{"good time", args{"1534552745.065949"}, time.Date(2018, 8, 18, 0, 39, 05, 65949, time.UTC), false},
		{"time without millis", args{"0"}, time.Date(1970, 1, 1, 0, 00, 00, 0, time.UTC), false},
		{"invalid time", args{"x"}, time.Time{}, true},
		{"invalid time", args{"x.x"}, time.Time{}, true},
		{"invalid time", args{"4.x"}, time.Time{}, true},
		{"invalid time", args{"x.4"}, time.Time{}, true},
		{"invalid time", args{".4"}, time.Time{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fromSlackTime(tt.args.timestamp)
			if (err != nil) != tt.wantErr {
				t.Errorf("fromSlackTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fromSlackTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

const twoJSONMessages = `[{
	"type": "message",
	"user": "U0LKLSNER",
	"text": ":-)",
	"ts": "1501195054.703005",
	"replace_original": false,
	"delete_original": false
},{
	"type": "message",
	"user": "U0LKLSNER",
	"ts": "1520889249.000069",
	"files": [
		{
			"id": "F9NKY5WLU",
			"created": 1520889249,
			"timestamp": 1520889249,
			"name": "Image uploaded from iOS.jpg",
			"title": "Paris, date unknown",
			"mimetype": "image/jpeg",
			"image_exif_rotation": 1,
			"filetype": "jpg",
			"pretty_type": "JPEG",
			"user": "U0LKLSNER",
			"mode": "hosted",
			"editable": false,
			"is_external": false,
			"external_type": "",
			"size": 260015,
			"url": "",
			"url_download": "",
			"url_private": "https://files.slack.com/files-pri/T03K8JZAN-F9NKY5WLU/image_uploaded_from_ios.jpg",
			"url_private_download": "https://files.slack.com/files-pri/T03K8JZAN-F9NKY5WLU/download/image_uploaded_from_ios.jpg",
			"original_h": 718,
			"original_w": 599,
			"thumb_64": "https://files.slack.com/files-tmb/T03K8JZAN-F9NKY5WLU-58ea730fe0/image_uploaded_from_ios_64.jpg",
			"thumb_80": "https://files.slack.com/files-tmb/T03K8JZAN-F9NKY5WLU-58ea730fe0/image_uploaded_from_ios_80.jpg",
			"thumb_160": "https://files.slack.com/files-tmb/T03K8JZAN-F9NKY5WLU-58ea730fe0/image_uploaded_from_ios_160.jpg",
			"thumb_360": "https://files.slack.com/files-tmb/T03K8JZAN-F9NKY5WLU-58ea730fe0/image_uploaded_from_ios_360.jpg",
			"thumb_360_gif": "",
			"thumb_360_w": 300,
			"thumb_360_h": 360,
			"thumb_480": "https://files.slack.com/files-tmb/T03K8JZAN-F9NKY5WLU-58ea730fe0/image_uploaded_from_ios_480.jpg",
			"thumb_480_w": 400,
			"thumb_480_h": 480,
			"thumb_720": "",
			"thumb_720_w": 0,
			"thumb_720_h": 0,
			"thumb_960": "",
			"thumb_960_w": 0,
			"thumb_960_h": 0,
			"thumb_1024": "",
			"thumb_1024_w": 0,
			"thumb_1024_h": 0,
			"permalink": "https://dbvisit.slack.com/files/U0LKLSNER/F9NKY5WLU/image_uploaded_from_ios.jpg",
			"permalink_public": "https://slack-files.com/T03K8JZAN-F9NKY5WLU-68449da10f",
			"edit_link": "",
			"preview": "",
			"preview_highlight": "",
			"lines": 0,
			"lines_more": 0,
			"is_public": false,
			"public_url_shared": false,
			"channels": null,
			"groups": null,
			"ims": null,
			"initial_comment": {},
			"comments_count": 0,
			"num_stars": 0,
			"is_starred": false
		}
	],
	"upload": true,
	"replace_original": false,
	"delete_original": false
}]`
