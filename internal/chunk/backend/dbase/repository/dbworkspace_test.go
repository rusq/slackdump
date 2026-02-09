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

package repository

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v4/internal/chunk"
)

var wsp1 = &slack.AuthTestResponse{
	URL:    "http://lol.slack.com",
	Team:   "lolzteam",
	User:   "lolzuser",
	TeamID: "T123456",
	UserID: "U123456",
}

var wsp1_ = &slack.AuthTestResponse{
	URL:    wsp1.URL,
	Team:   wsp1.Team,
	User:   "renameduser",
	TeamID: wsp1.TeamID,
	UserID: wsp1.UserID,
}

func Test_workspaceRepository_GetWorkspace(t *testing.T) {
	var (
		dbwsp1, _ = NewDBWorkspace(1, wsp1)
		dbwsp2, _ = NewDBWorkspace(2, wsp1_)
	)
	type fields struct {
		genericRepository genericRepository[DBWorkspace]
	}
	type args struct {
		ctx  context.Context
		conn sqlx.QueryerContext
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFn
		want    DBWorkspace
		wantErr bool
	}{
		{
			name: "returns the latest version of the workspace",
			fields: fields{
				genericRepository: genericRepository[DBWorkspace]{DBWorkspace{}},
			},
			args: args{
				ctx:  t.Context(),
				conn: testConn(t),
			},
			prepFn: func(t *testing.T, conn PrepareExtContext) {
				t.Helper()
				prepChunk(chunk.CWorkspaceInfo, chunk.CWorkspaceInfo)(t, conn)
				wr := NewWorkspaceRepository()
				if err := wr.Insert(t.Context(), conn, dbwsp1, dbwsp2); err != nil {
					t.Fatal(err)
				}
			},
			want: *dbwsp2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.conn.(PrepareExtContext))
			}
			r := workspaceRepository{
				genericRepository: tt.fields.genericRepository,
			}
			got, err := r.GetWorkspace(tt.args.ctx, tt.args.conn)
			if (err != nil) != tt.wantErr {
				t.Errorf("workspaceRepository.GetWorkspace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewDBWorkspace(t *testing.T) {
	type args struct {
		chunkID int64
		wi      *slack.AuthTestResponse
	}
	tests := []struct {
		name    string
		args    args
		want    *DBWorkspace
		wantErr bool
	}{
		{
			name: "creates a new DBWorkspace",
			args: args{
				chunkID: 42,
				wi:      wsp1,
			},
			want: &DBWorkspace{
				ID:           0,
				ChunkID:      42,
				Team:         wsp1.Team,
				User:         ptr(wsp1.User),
				TeamID:       "T123456",
				UserID:       "U123456",
				EnterpriseID: nil,
				URL:          "http://lol.slack.com",
				Data:         must(marshal(wsp1)),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDBWorkspace(tt.args.chunkID, tt.args.wi)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDBWorkspace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
