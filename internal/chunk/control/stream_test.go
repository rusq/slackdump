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

package control

import (
	"context"
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v4/internal/chunk/control/mock_control"
	"github.com/rusq/slackdump/v4/mocks/mock_processor"
)

func Test_userCollectingStreamer_Users(t *testing.T) {
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()

	type fields struct {
		// Streamer Streamer
		userIDC       <-chan []string
		includeLabels bool
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
				ctx: t.Context(),
			},
			prepFn: func(f *fields) {
				userIDC := make(chan []string, 1)
				defer close(userIDC)
				f.userIDC = userIDC
				userIDC <- []string{"U12345678", "U87654321"}
			},
			expectFn: func(ms *mock_control.MockStreamer, mup *mock_processor.MockUsers) {
				ms.EXPECT().UsersBulkWithCustom(gomock.Any(), mup, false, "U12345678", "U87654321").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "propagates include labels",
			fields: fields{
				includeLabels: true,
			},
			args: args{
				ctx: t.Context(),
			},
			prepFn: func(f *fields) {
				userIDC := make(chan []string, 1)
				defer close(userIDC)
				f.userIDC = userIDC
				userIDC <- []string{"U12345678", "U87654321"}
			},
			expectFn: func(ms *mock_control.MockStreamer, mup *mock_processor.MockUsers) {
				ms.EXPECT().UsersBulkWithCustom(gomock.Any(), mup, true, "U12345678", "U87654321").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "method returns an error",
			args: args{
				ctx: t.Context(),
			},
			prepFn: func(f *fields) {
				userIDC := make(chan []string, 1)
				defer close(userIDC)
				f.userIDC = userIDC
				userIDC <- []string{"U12345678", "U87654321"}
			},
			expectFn: func(ms *mock_control.MockStreamer, mup *mock_processor.MockUsers) {
				ms.EXPECT().UsersBulkWithCustom(gomock.Any(), mup, false, "U12345678", "U87654321").Return(assert.AnError)
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
				Streamer:      ms,
				userIDC:       tt.fields.userIDC,
				includeLabels: tt.fields.includeLabels,
			}
			if err := u.Users(tt.args.ctx, mup, tt.args.opt...); (err != nil) != tt.wantErr {
				t.Errorf("userCollectingStreamer.Users() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
