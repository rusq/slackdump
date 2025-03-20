package control

import (
	"context"
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v3/internal/chunk/control/mock_control"
	"github.com/rusq/slackdump/v3/mocks/mock_processor"
)

func Test_userCollectingStreamer_Users(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	type fields struct {
		// Streamer Streamer
		userIDC <-chan []string
	}
	type args struct {
		ctx context.Context
		// proc processor.Users
		opt []slack.GetUsersOption
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		prepFn   func(f *fields)
		expectFn func(ms *mock_control.MockStreamer, mup *mock_processor.MockUsers)
		wantErr  bool
	}{
		{
			name: "cancelled context",
			args: args{
				ctx: cancelled,
			},
			wantErr: true,
		},
		{
			name: "test User IDs",
			args: args{
				ctx: context.Background(),
			},
			prepFn: func(f *fields) {
				userIDC := make(chan []string, 1)
				defer close(userIDC)
				f.userIDC = userIDC
				userIDC <- []string{"U12345678", "U87654321"}
			},
			expectFn: func(ms *mock_control.MockStreamer, mup *mock_processor.MockUsers) {
				ms.EXPECT().UsersBulk(gomock.Any(), mup, "U12345678", "U87654321").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "method returns an error",
			args: args{
				ctx: context.Background(),
			},
			prepFn: func(f *fields) {
				userIDC := make(chan []string, 1)
				defer close(userIDC)
				f.userIDC = userIDC
				userIDC <- []string{"U12345678", "U87654321"}
			},
			expectFn: func(ms *mock_control.MockStreamer, mup *mock_processor.MockUsers) {
				ms.EXPECT().UsersBulk(gomock.Any(), mup, "U12345678", "U87654321").Return(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ms := mock_control.NewMockStreamer(ctrl)
			mup := mock_processor.NewMockUsers(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(ms, mup)
			}
			if tt.prepFn != nil {
				tt.prepFn(&tt.fields)
			}
			u := &userCollectingStreamer{
				Streamer: ms,
				userIDC:  tt.fields.userIDC,
			}
			if err := u.Users(tt.args.ctx, mup, tt.args.opt...); (err != nil) != tt.wantErr {
				t.Errorf("userCollectingStreamer.Users() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
