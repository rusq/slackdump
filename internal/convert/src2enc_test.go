package convert

import (
	"context"
	"testing"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/source"
	"github.com/rusq/slackdump/v3/processor"
)

func Test_encodeMessages(t *testing.T) {
	type args struct {
		ctx context.Context
		rec processor.Conversations
		src source.Sourcer
		ch  *slack.Channel
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := encodeMessages(tt.args.ctx, tt.args.rec, tt.args.src, tt.args.ch); (err != nil) != tt.wantErr {
				t.Errorf("encodeMessages() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
