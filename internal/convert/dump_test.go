// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
package convert

import (
	"testing"

	"github.com/rusq/slack"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v4/internal/convert/mock_convert"
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
