package base

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestYesNoWR(t *testing.T) {
	type args struct {
		r       io.Reader
		message string
	}
	tests := []struct {
		name  string
		args  args
		want  bool
		wantW string
	}{
		{
			name: "yes",
			args: args{
				r:       strings.NewReader("y\n"),
				message: "message",
			},
			want:  true,
			wantW: "message? (y/N) ",
		},
		{
			name: "no",
			args: args{
				r:       strings.NewReader("n\n"),
				message: "message",
			},
			want:  false,
			wantW: "message? (y/N) ",
		},
		{
			name: "any other key",
			args: args{
				r:       strings.NewReader("x\nn\n"),
				message: "message",
			},
			want:  false,
			wantW: "message? (y/N) Please answer yes or no and press Enter or Return.\nmessage? (y/N) ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if got := YesNoWR(w, tt.args.r, tt.args.message); got != tt.want {
				t.Errorf("YesNoWR() = %v, want %v", got, tt.want)
			}
			if gotW := w.String(); !assert.Equal(t, tt.wantW, gotW) {
				t.Errorf("YesNoWR() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}
