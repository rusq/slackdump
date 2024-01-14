package chunk

import (
	"reflect"
	"testing"

	"github.com/rusq/slackdump/v3/internal/structures"
)

func TestToFileID(t *testing.T) {
	type args struct {
		channelID     string
		threadTS      string
		includeThread bool
	}
	tests := []struct {
		name string
		args args
		want FileID
	}{
		{
			"just a channel",
			args{
				channelID:     "C12345678",
				threadTS:      "",
				includeThread: false,
			},
			"C12345678",
		},
		{
			"channel and thread",
			args{
				channelID:     "C12345678",
				threadTS:      "12345678.123456",
				includeThread: true,
			},
			"C12345678-12345678.123456",
		},
		{
			"channel and empty thread",
			args{
				channelID:     "C12345678",
				threadTS:      "",
				includeThread: true,
			},
			"C12345678",
		},
		{
			"channel and thread, but includeThread is false",
			args{
				channelID:     "C12345678",
				threadTS:      "12345678.123456",
				includeThread: false,
			},
			"C12345678",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToFileID(tt.args.channelID, tt.args.threadTS, tt.args.includeThread); got != tt.want {
				t.Errorf("ToFileID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLinkToFileID(t *testing.T) {
	type args struct {
		sl            structures.SlackLink
		includeThread bool
	}
	tests := []struct {
		name string
		args args
		want FileID
	}{
		{
			"just a channel",
			args{
				sl:            structures.SlackLink{Channel: "C12345678"},
				includeThread: false,
			},
			"C12345678",
		},
		{
			"channel and thread",
			args{
				sl:            structures.SlackLink{Channel: "C12345678", ThreadTS: "12345678.123456"},
				includeThread: true,
			},
			"C12345678-12345678.123456",
		},
		{
			"channel and empty thread",
			args{
				sl:            structures.SlackLink{Channel: "C12345678"},
				includeThread: true,
			},
			"C12345678",
		},
		{
			"channel and thread, but includeThread is false",
			args{
				sl:            structures.SlackLink{Channel: "C12345678", ThreadTS: "12345678.123456"},
				includeThread: false,
			},
			"C12345678",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LinkToFileID(tt.args.sl, tt.args.includeThread); got != tt.want {
				t.Errorf("LinkToFileID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileID_Split(t *testing.T) {
	tests := []struct {
		name          string
		id            FileID
		wantChannelID string
		wantThreadTS  string
	}{
		{
			"just a channel",
			"C12345678",
			"C12345678",
			"",
		},
		{
			"channel and thread",
			"C12345678-12345678.123456",
			"C12345678",
			"12345678.123456",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotChannelID, gotThreadTS := tt.id.Split()
			if gotChannelID != tt.wantChannelID {
				t.Errorf("FileID.Split() gotChannelID = %v, want %v", gotChannelID, tt.wantChannelID)
			}
			if gotThreadTS != tt.wantThreadTS {
				t.Errorf("FileID.Split() gotThreadTS = %v, want %v", gotThreadTS, tt.wantThreadTS)
			}
		})
	}
}

func TestFileID_SlackLink(t *testing.T) {
	tests := []struct {
		name string
		id   FileID
		want structures.SlackLink
	}{
		{
			"just a channel",
			"C12345678",
			structures.SlackLink{Channel: "C12345678"},
		},
		{
			"channel and thread",
			"C12345678-12345678.123456",
			structures.SlackLink{Channel: "C12345678", ThreadTS: "12345678.123456"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.SlackLink(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FileID.SlackLink() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileID_String(t *testing.T) {
	tests := []struct {
		name string
		id   FileID
		want string
	}{
		{
			"just a channel",
			"C12345678",
			"C12345678",
		},
		{
			"channel and thread",
			"C12345678-12345678.123456",
			"C12345678-12345678.123456",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.String(); got != tt.want {
				t.Errorf("FileID.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
