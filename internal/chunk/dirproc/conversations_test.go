package dirproc

import (
	"context"
	"testing"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/logger"
	"github.com/rusq/slackdump/v3/processor"
)

func TestConversations_Messages(t *testing.T) {
	type fields struct {
		dir         *chunk.Directory
		t           *filetracker
		lg          logger.Interface
		subproc     processor.Filer
		recordFiles bool
		tf          Transformer
	}
	type args struct {
		ctx        context.Context
		channelID  string
		numThreads int
		isLast     bool
		mm         []slack.Message
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cv := &Conversations{
				dir:         tt.fields.dir,
				t:           tt.fields.t,
				lg:          tt.fields.lg,
				subproc:     tt.fields.subproc,
				recordFiles: tt.fields.recordFiles,
				tf:          tt.fields.tf,
			}
			if err := cv.Messages(tt.args.ctx, tt.args.channelID, tt.args.numThreads, tt.args.isLast, tt.args.mm); (err != nil) != tt.wantErr {
				t.Errorf("Conversations.Messages() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
