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
package slackdump

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v4/internal/client/mock_client"
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
