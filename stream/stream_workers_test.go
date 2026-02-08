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
package stream

import (
	"context"
	"errors"
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v3/internal/client/mock_client"
	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/rusq/slackdump/v3/mocks/mock_processor"
)

func TestStream_canvas(t *testing.T) {
	testChannel := fixtures.Load[[]*slack.Channel](fixtures.TestChannelsJSON)[0]
	type args struct {
		ctx context.Context
		// proc    processor.Conversations
		channel *slack.Channel
		fileId  string
	}
	tests := []struct {
		name     string
		fields   *Stream
		args     args
		expectFn func(ms *mock_client.MockSlack, mc *mock_processor.MockConversations)
		wantErr  bool
	}{
		{
			name:   "file ID is empty",
			fields: &Stream{},
			args: args{
				ctx:     t.Context(),
				channel: &slack.Channel{},
				fileId:  "",
			},
			wantErr: false,
		},
		{
			name:   "getfileinfocontext returns an error",
			fields: &Stream{},
			args: args{
				ctx:    t.Context(),
				fileId: "F123456",
			},
			expectFn: func(ms *mock_client.MockSlack, mc *mock_processor.MockConversations) {
				ms.EXPECT().GetFileInfoContext(gomock.Any(), "F123456", 0, 1).Return(nil, nil, nil, errors.New("getfileinfocontext error"))
			},
			wantErr: true,
		},
		{
			name:   "file not found",
			fields: &Stream{},
			args: args{
				ctx:    t.Context(),
				fileId: "F123456",
			},
			expectFn: func(ms *mock_client.MockSlack, mc *mock_processor.MockConversations) {
				ms.EXPECT().GetFileInfoContext(gomock.Any(), "F123456", 0, 1).Return(nil, nil, nil, nil)
			},
			wantErr: true,
		},
		{
			name:   "success",
			fields: &Stream{},
			args: args{
				ctx:     t.Context(),
				channel: testChannel,
				fileId:  "F123456",
			},
			expectFn: func(ms *mock_client.MockSlack, mc *mock_processor.MockConversations) {
				ms.EXPECT().
					GetFileInfoContext(gomock.Any(), "F123456", 0, 1).
					Return(&slack.File{ID: "F123456"}, nil, nil, nil)
				mc.EXPECT().
					Files(gomock.Any(), testChannel, slack.Message{}, []slack.File{{ID: "F123456"}}).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "processor returns an error",
			fields: &Stream{},
			args: args{
				ctx:     t.Context(),
				channel: testChannel,
				fileId:  "F123456",
			},
			expectFn: func(ms *mock_client.MockSlack, mc *mock_processor.MockConversations) {
				ms.EXPECT().
					GetFileInfoContext(gomock.Any(), "F123456", 0, 1).
					Return(&slack.File{ID: "F123456"}, nil, nil, nil)
				mc.EXPECT().
					Files(gomock.Any(), testChannel, slack.Message{}, []slack.File{{ID: "F123456"}}).
					Return(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ms := mock_client.NewMockSlack(ctrl)
			mc := mock_processor.NewMockConversations(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(ms, mc)
			}
			cs := tt.fields
			cs.client = ms
			if err := cs.canvas(tt.args.ctx, mc, tt.args.channel, tt.args.fileId); (err != nil) != tt.wantErr {
				t.Errorf("Stream.canvas() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
