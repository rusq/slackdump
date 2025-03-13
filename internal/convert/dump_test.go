package convert

import (
	"testing"

	"github.com/rusq/slack"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v3/internal/convert/mock_convert"
)

func Test_fileHandler_copyFiles(t *testing.T) {
	type args struct {
		channelID string
		in1       string
		mm        []slack.Message
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(mc *mock_convert.Mockcopier)
		wantErr  bool
	}{
		{
			name: "Test 1",
			args: args{
				channelID: "channelID",
				mm: []slack.Message{
					{Msg: slack.Msg{Files: []slack.File{
						{ID: "F11111111", Name: "name.ext"},
						{ID: "F22222222", Name: "name.ext"},
					}}},
					{Msg: slack.Msg{Files: []slack.File{
						{ID: "F33333333", Name: "name.ext"},
						{ID: "F44444444", Name: "name.ext"},
					}}},
				},
			},
			expectFn: func(mc *mock_convert.Mockcopier) {
				mc.EXPECT().Copy(gomock.Any(), gomock.Any()).Return(nil).Times(2) // 1 for each msg
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mc := mock_convert.NewMockcopier(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(mc)
			}
			f := &fileHandler{
				fc: mc,
			}
			if err := f.copyFiles(tt.args.channelID, tt.args.in1, tt.args.mm); (err != nil) != tt.wantErr {
				t.Errorf("fileHandler.copyFiles() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
