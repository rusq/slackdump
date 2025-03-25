package resume

import (
	"context"
	_ "embed"
	"testing"
	"time"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/rusq/slackdump/v3/internal/source"
	"github.com/rusq/slackdump/v3/internal/source/mock_source"
	"github.com/rusq/slackdump/v3/internal/structures"
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
				ctx: context.Background(),
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
				ctx: context.Background(),
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
				ctx: context.Background(),
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
				ctx: context.Background(),
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
				ctx: context.Background(),
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
				ctx: context.Background(),
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
				ctx: context.Background(),
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
				ctx: context.Background(),
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
				ctx: context.Background(),
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
				ctx: context.Background(),
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
				ctx: context.Background(),
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
				ctx: context.Background(),
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
				ctx: context.Background(),
			},
			expectFn: func(ms *mock_source.MockSourcer) {
				ms.EXPECT().Users(gomock.Any()).Return([]slack.User{}, nil)
			},
			wantErr: true,
		},
		{
			name: "fixture users",
			args: args{
				ctx: context.Background(),
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
				ctx: context.Background(),
			},
			expectFn: func(ms *mock_source.MockSourcer) {
				ms.EXPECT().Users(gomock.Any()).Return(nil, assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "unable to reliably determine the team ID",
			args: args{
				ctx: context.Background(),
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
				ctx:            context.Background(),
				includeThreads: false,
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
				ctx:            context.Background(),
				includeThreads: false,
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
				ctx:            context.Background(),
				includeThreads: false,
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
				ctx:            context.Background(),
				includeThreads: true,
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
				ctx:            context.Background(),
				includeThreads: false,
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
			got, err := latest(tt.args.ctx, mr, tt.args.includeThreads)
			if (err != nil) != tt.wantErr {
				t.Errorf("latest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
