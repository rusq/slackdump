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
	"reflect"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/rusq/slackdump/v4/internal/chunk"
	"github.com/rusq/slackdump/v4/internal/testutil"
)

func TestNewDBChannelUser(t *testing.T) {
	type args struct {
		chunkID   int64
		n         int
		channelID string
		userID    string
	}
	tests := []struct {
		name    string
		args    args
		want    *DBChannelUser
		wantErr bool
	}{
		{
			name: "creates a new DBChannelUser",
			args: args{
				chunkID:   1,
				n:         50,
				channelID: "C100",
				userID:    "U100",
			},
			want: &DBChannelUser{
				UserID:    "U100",
				ChunkID:   1,
				ChannelID: "C100",
				Index:     50,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDBChannelUser(tt.args.chunkID, tt.args.n, tt.args.channelID, tt.args.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDBChannelUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDBChannelUser() = %v, want %v", got, tt.want)
			}
		})
	}
}

var (
	testC1U1, _ = NewDBChannelUser(1, 0, "C111", "UAAA")
	testC1U2, _ = NewDBChannelUser(1, 1, "C111", "UBBB")
	testC1U3, _ = NewDBChannelUser(1, 2, "C111", "UCCC")
	testC2U4, _ = NewDBChannelUser(2, 0, "C222", "UDDD")
	testC2U1, _ = NewDBChannelUser(2, 1, "C222", "UAAA")

	// C333 is a mutation test
	testC3U5, _ = NewDBChannelUser(3, 0, "C333", "UEEE")
	// Later chunk for C333 has different users in the channel, i.e.
	// UEEE left, and UAAA, UDDD joined.
	testC3_U1, _ = NewDBChannelUser(4, 0, "C333", "UAAA")
	testC3_U4, _ = NewDBChannelUser(4, 1, "C333", "UDDD")
)

func prepChannelUsers(t *testing.T, db PrepareExtContext) {
	prepChunk(chunk.CChannelUsers, chunk.CChannelUsers, chunk.CChannelUsers, chunk.CChannelUsers)(t, db)

	cur := NewChannelUserRepository()
	if err := cur.Insert(t.Context(), db, testC1U1, testC1U2, testC1U3, testC2U4, testC2U1, testC3U5, testC3_U1, testC3_U4); err != nil {
		t.Fatalf("prepChannelUsers: %v", err)
	}
}

func Test_channelUserRepository_GetByChannelID(t *testing.T) {
	type fields struct {
		genericRepository genericRepository[DBChannelUser]
	}
	type args struct {
		ctx       context.Context
		db        sqlx.QueryerContext
		channelID string
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		prepareFn utilityFn
		want      []testutil.TestResult[DBChannelUser]
		wantErr   bool
	}{
		{
			name: "returns users for channel C111",
			fields: fields{
				genericRepository: newGenericRepository(DBChannelUser{}),
			},
			args: args{
				ctx:       t.Context(),
				db:        testConn(t),
				channelID: "C111",
			},
			prepareFn: prepChannelUsers,
			want: []testutil.TestResult[DBChannelUser]{
				{V: *testC1U1},
				{V: *testC1U2},
				{V: *testC1U3},
			},
		},
		{
			name: "returns users for channel C222, in order",
			fields: fields{
				genericRepository: newGenericRepository(DBChannelUser{}),
			},
			args: args{
				ctx:       t.Context(),
				db:        testConn(t),
				channelID: "C222",
			},
			prepareFn: prepChannelUsers,
			want: []testutil.TestResult[DBChannelUser]{
				{V: *testC2U1},
				{V: *testC2U4},
			},
		},
		{
			name: "returns empty set for missing channel",
			fields: fields{
				genericRepository: newGenericRepository(DBChannelUser{}),
			},
			args: args{
				ctx:       t.Context(),
				db:        testConn(t),
				channelID: "C---",
			},
			prepareFn: prepChannelUsers,
			want:      nil,
			wantErr:   false,
		},
		{
			name: "handles latest state for the C333",
			fields: fields{
				genericRepository: newGenericRepository(DBChannelUser{}),
			},
			args: args{
				ctx:       t.Context(),
				db:        testConn(t),
				channelID: "C333",
			},
			prepareFn: prepChannelUsers,
			want: []testutil.TestResult[DBChannelUser]{
				{V: *testC3_U1},
				{V: *testC3_U4},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepareFn != nil {
				tt.prepareFn(t, tt.args.db.(PrepareExtContext))
			}
			r := channelUserRepository{
				genericRepository: tt.fields.genericRepository,
			}
			got, err := r.GetByChannelID(tt.args.ctx, tt.args.db, tt.args.channelID)
			if (err != nil) != tt.wantErr {
				t.Errorf("channelUserRepository.GetByChannelID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			testutil.AssertIterResult(t, tt.want, got)
		})
	}
}
