package slackdump

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v3/internal/client/mock_client"
)

func TestSession_DumpEmojis(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name     string
		args     args
		expectfn func(m *mock_client.MockSlackClienter)
		want     map[string]string
		wantErr  bool
	}{
		{
			"ok",
			args{t.Context()},
			func(m *mock_client.MockSlackClienter) {
				m.EXPECT().
					GetEmojiContext(gomock.Any()).
					Return(map[string]string{"foo": "bar"}, nil)
			},
			map[string]string{"foo": "bar"},
			false,
		},
		{
			"error is propagated",
			args{t.Context()},
			func(m *mock_client.MockSlackClienter) {
				m.EXPECT().
					GetEmojiContext(gomock.Any()).
					Return(nil, errors.New("not today sir"))
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcl := mock_client.NewMockSlackClienter(gomock.NewController(t))
			tt.expectfn(mcl)
			s := &Session{
				client: mcl,
			}
			got, err := s.DumpEmojis(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Session.DumpEmojis() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Session.DumpEmojis() = %v, want %v", got, tt.want)
			}
		})
	}
}
