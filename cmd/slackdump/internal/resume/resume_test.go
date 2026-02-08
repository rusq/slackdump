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
package resume

import (
	"context"
	_ "embed"
	"testing"
	"time"

	"github.com/rusq/slack"
	"github.com/sosodev/duration"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v4"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/internal/fixtures"
	"github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/source"
	"github.com/rusq/slackdump/v4/source/mock_source"
)

func Test_ensureSameWorkspace(t *testing.T) {
	type args struct {
		ctx context.Context
		// src  source.Sourcer
		info *slackdump.WorkspaceInfo
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(ms *mock_source.MockSourcer)
		wantErr  bool
	}{
		{
			name: "match",
			args: args{
				ctx: t.Context(),
				info: &slackdump.WorkspaceInfo{
					TeamID: "T123",
				},
			},
			expectFn: func(ms *mock_source.MockSourcer) {
				ms.EXPECT().WorkspaceInfo(gomock.Any()).Return(&slackdump.WorkspaceInfo{
					TeamID: "T123",
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "mismatch",
			args: args{
				ctx: t.Context(),
				info: &slackdump.WorkspaceInfo{
					TeamID: "T123",
				},
			},
			expectFn: func(ms *mock_source.MockSourcer) {
				ms.EXPECT().WorkspaceInfo(gomock.Any()).Return(&slackdump.WorkspaceInfo{
					TeamID: "T456",
				}, nil)
			},
			wantErr: true,
		},
		{
			name: "error",
			args: args{
				ctx: t.Context(),
				info: &slackdump.WorkspaceInfo{
					TeamID: "T123",
				},
			},
			expectFn: func(ms *mock_source.MockSourcer) {
				ms.EXPECT().WorkspaceInfo(gomock.Any()).Return(nil, assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "no workspace info, no users, no channels",
			args: args{
				ctx: t.Context(),
				info: &slackdump.WorkspaceInfo{
					TeamID: "T123",
				},
			},
			expectFn: func(ms *mock_source.MockSourcer) {
				ms.EXPECT().WorkspaceInfo(gomock.Any()).Return(nil, source.ErrNotFound)
				ms.EXPECT().Users(gomock.Any()).Return([]slack.User{}, source.ErrNotFound)
				ms.EXPECT().Channels(gomock.Any()).Return([]slack.Channel{}, source.ErrNotFound)
			},
			wantErr: true,
		},
		{
			name: "no workspace info, no users, fixture channels, workspace mismatch",
			args: args{
				ctx: t.Context(),
				info: &slackdump.WorkspaceInfo{
					TeamID: "T123",
				},
			},
			expectFn: func(ms *mock_source.MockSourcer) {
				ms.EXPECT().WorkspaceInfo(gomock.Any()).Return(nil, source.ErrNotFound)
				ms.EXPECT().Users(gomock.Any()).Return([]slack.User{}, source.ErrNotFound)
				channels := fixtures.Load[[]slack.Channel](fixtures.TestChannelsWithTeamJSON)
				ms.EXPECT().Channels(gomock.Any()).Return(channels, nil)
			},
			wantErr: true,
		},
		{
			name: "no workspace info, no users, fixture channels, workspace match",
			args: args{
				ctx: t.Context(),
				info: &slackdump.WorkspaceInfo{
					TeamID: "THY5HTZ8U",
				},
			},
			expectFn: func(ms *mock_source.MockSourcer) {
				ms.EXPECT().WorkspaceInfo(gomock.Any()).Return(nil, source.ErrNotFound)
				ms.EXPECT().Users(gomock.Any()).Return([]slack.User{}, source.ErrNotFound)
				channels := fixtures.Load[[]slack.Channel](fixtures.TestChannelsWithTeamJSON)
				ms.EXPECT().Channels(gomock.Any()).Return(channels, nil)
			},
			wantErr: false,
		},
		{
			name: "no workspace info, fixture users",
			args: args{
				ctx: t.Context(),
				info: &slackdump.WorkspaceInfo{
					TeamID: "TFCSDNRL5",
				},
			},
			expectFn: func(ms *mock_source.MockSourcer) {
				ms.EXPECT().WorkspaceInfo(gomock.Any()).Return(nil, source.ErrNotFound)
				users := fixtures.Load[[]slack.User](string(fixtures.TestExpUsersJSON))
				ms.EXPECT().Users(gomock.Any()).Return(users, nil)
			},
			wantErr: false,
		},
		{
			name: "no workspace info, fixture users, workspace mismatch",
			args: args{
				ctx: t.Context(),
				info: &slackdump.WorkspaceInfo{
					TeamID: "T123",
				},
			},
			expectFn: func(ms *mock_source.MockSourcer) {
				ms.EXPECT().WorkspaceInfo(gomock.Any()).Return(nil, source.ErrNotFound)
				users := fixtures.Load[[]slack.User](string(fixtures.TestExpUsersJSON))
				ms.EXPECT().Users(gomock.Any()).Return(users, nil)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ms := mock_source.NewMockSourcer(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(ms)
			}
			if err := ensureSameWorkspace(tt.args.ctx, ms, tt.args.info); (err != nil) != tt.wantErr {
				t.Errorf("ensureSameWorkspace() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_channelTeam(t *testing.T) {
	type args struct {
		ctx context.Context
		// src source.Sourcer
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(ms *mock_source.MockSourcer)
		want     string
		wantErr  bool
	}{
		{
			name: "no channels",
			args: args{
				ctx: t.Context(),
			},
			expectFn: func(ms *mock_source.MockSourcer) {
				ms.EXPECT().Channels(gomock.Any()).Return([]slack.Channel{}, nil)
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "fixture channels",
			args: args{
				ctx: t.Context(),
			},
			expectFn: func(ms *mock_source.MockSourcer) {
				channels := fixtures.Load[[]slack.Channel](fixtures.TestChannelsWithTeamJSON)
				ms.EXPECT().Channels(gomock.Any()).Return(channels, nil)
			},
			want:    "THY5HTZ8U",
			wantErr: false,
		},
		{
			name: "API error",
			args: args{
				ctx: t.Context(),
			},
			expectFn: func(ms *mock_source.MockSourcer) {
				ms.EXPECT().Channels(gomock.Any()).Return(nil, assert.AnError)
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "fixture channels, no team ID",
			args: args{
				ctx: t.Context(),
			},
			expectFn: func(ms *mock_source.MockSourcer) {
				channels := fixtures.Load[[]slack.Channel](fixtures.TestChannelsJSON)
				ms.EXPECT().Channels(gomock.Any()).Return(channels, nil)
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ms := mock_source.NewMockSourcer(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(ms)
			}
			got, err := channelsTeam(tt.args.ctx, ms)
			if (err != nil) != tt.wantErr {
				t.Errorf("channelTeam() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("channelTeam() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_usersTeam(t *testing.T) {
	type args struct {
		ctx context.Context
		// src source.Sourcer
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(ms *mock_source.MockSourcer)
		want     string
		wantErr  bool
	}{
		{
			name: "no users",
			args: args{
				ctx: t.Context(),
			},
			expectFn: func(ms *mock_source.MockSourcer) {
				ms.EXPECT().Users(gomock.Any()).Return([]slack.User{}, nil)
			},
			wantErr: true,
		},
		{
			name: "fixture users",
			args: args{
				ctx: t.Context(),
			},
			expectFn: func(ms *mock_source.MockSourcer) {
				users := fixtures.Load[[]slack.User](string(fixtures.TestExpUsersJSON))
				ms.EXPECT().Users(gomock.Any()).Return(users, nil)
			},
			want:    "TFCSDNRL5",
			wantErr: false,
		},
		{
			name: "API error",
			args: args{
				ctx: t.Context(),
			},
			expectFn: func(ms *mock_source.MockSourcer) {
				ms.EXPECT().Users(gomock.Any()).Return(nil, assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "unable to reliably determine the team ID",
			args: args{
				ctx: t.Context(),
			},
			expectFn: func(ms *mock_source.MockSourcer) {
				users := []slack.User{
					{ID: "U123", TeamID: "T123"},
					{ID: "U456", TeamID: "T456"},
				}
				ms.EXPECT().Users(gomock.Any()).Return(users, nil)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ms := mock_source.NewMockSourcer(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(ms)
			}
			got, err := usersTeam(tt.args.ctx, ms)
			if (err != nil) != tt.wantErr {
				t.Errorf("usersTeam() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("usersTeam() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_latest(t *testing.T) {
	type args struct {
		ctx context.Context
		// src source.Resumer
		includeThreads bool
		lookBack       time.Duration
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(mr *mock_source.MockResumer)
		want     *structures.EntityList
		wantErr  bool
	}{
		{
			name: "resumer error",
			args: args{
				ctx:            t.Context(),
				includeThreads: false,
				lookBack:       0,
			},
			expectFn: func(mr *mock_source.MockResumer) {
				mr.EXPECT().Latest(gomock.Any()).Return(nil, assert.AnError)
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "no entities",
			args: args{
				ctx:            t.Context(),
				includeThreads: false,
				lookBack:       0,
			},
			expectFn: func(mr *mock_source.MockResumer) {
				mr.EXPECT().Latest(gomock.Any()).Return(map[structures.SlackLink]time.Time{}, nil)
			},
			want:    &structures.EntityList{},
			wantErr: false,
		},
		{
			name: "returns latest status",
			args: args{
				ctx:            t.Context(),
				includeThreads: false,
				lookBack:       0,
			},
			expectFn: func(mr *mock_source.MockResumer) {
				mr.EXPECT().Latest(gomock.Any()).Return(map[structures.SlackLink]time.Time{
					{Channel: "C123"}: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				}, nil)
			},
			want: structures.NewEntityListFromItems(
				structures.EntityItem{Id: "C123", Oldest: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), Latest: time.Time(cfg.Latest), Include: true},
			),
			wantErr: false,
		},
		{
			name: "returns latest status with thread",
			args: args{
				ctx:            t.Context(),
				includeThreads: true,
				lookBack:       0,
			},
			expectFn: func(mr *mock_source.MockResumer) {
				mr.EXPECT().Latest(gomock.Any()).Return(map[structures.SlackLink]time.Time{
					{Channel: "C123"}:                      time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					{Channel: "C456", ThreadTS: "123.456"}: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				}, nil)
			},
			want: structures.NewEntityListFromItems(
				structures.EntityItem{Id: "C123", Oldest: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), Latest: time.Time(cfg.Latest), Include: true},
				structures.EntityItem{Id: "C456:123.456", Oldest: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC), Latest: time.Time(cfg.Latest), Include: true},
			),
			wantErr: false,
		},
		{
			name: "returns latest status with thread, but includeThreads is false",
			args: args{
				ctx:            t.Context(),
				includeThreads: false,
				lookBack:       0,
			},
			expectFn: func(mr *mock_source.MockResumer) {
				mr.EXPECT().Latest(gomock.Any()).Return(map[structures.SlackLink]time.Time{
					{Channel: "C123"}:                      time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					{Channel: "C456", ThreadTS: "123.456"}: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				}, nil)
			},
			want: structures.NewEntityListFromItems(
				structures.EntityItem{Id: "C123", Oldest: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), Latest: time.Time(cfg.Latest), Include: true},
			),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mr := mock_source.NewMockResumer(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(mr)
			}
			got, err := latest(tt.args.ctx, mr, tt.args.includeThreads, tt.args.lookBack)
			if (err != nil) != tt.wantErr {
				t.Errorf("latest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_extDuration_Set(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		d       *extDuration
		args    args
		wantErr bool
		want    string
	}{
		{
			name: "1 week, no P prefix",
			d:    new(extDuration),
			args: args{
				s: "1w5dt2h3m4s",
			},
			wantErr: false,
			want:    "p1w5dt2h3m4s",
		},
		{
			name: "1 week (ISO 8601 format)",
			d:    new(extDuration),
			args: args{
				s: "P1W",
			},
			wantErr: false,
			want:    "p1w",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.d.Set(tt.args.s); (err != nil) != tt.wantErr {
				t.Errorf("extDuration.Set() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.want, tt.d.String())
		})
	}
}

func Test_extDuration_String(t *testing.T) {
	tests := []struct {
		name string
		d    *extDuration
		want string
	}{
		{
			"formats a duration",
			(*extDuration)(duration.FromTimeDuration(7*24*time.Hour + 5*24*time.Hour + 2*time.Hour + 3*time.Minute + 4*time.Second)),
			"p1w5dt2h3m4s",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.String(); got != tt.want {
				t.Errorf("extDuration.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
