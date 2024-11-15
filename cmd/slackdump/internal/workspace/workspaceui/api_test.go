package workspaceui

import (
	"context"
	"testing"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/rusq/slackdump/v3/internal/mocks/mock_auth"
	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
)

func Test_createAndSelect(t *testing.T) {
	type args struct {
		ctx context.Context
		// m    manager
		// prov auth.Provider
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(mp *mock_auth.MockProvider, mm *Mockmanager)
		want     string
		wantErr  bool
	}{
		{
			name: "provider test fails",
			args: args{
				ctx: context.Background(),
			},
			expectFn: func(mp *mock_auth.MockProvider, mm *Mockmanager) {
				mp.EXPECT().Test(gomock.Any()).Return(nil, assert.AnError)
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "save provider fails",
			args: args{
				ctx: context.Background(),
			},
			expectFn: func(mp *mock_auth.MockProvider, mm *Mockmanager) {
				mp.EXPECT().Test(gomock.Any()).Return(fixtures.LoadPtr[slack.AuthTestResponse](string(fixtures.TestAuthTestInfo)), nil)
				mm.EXPECT().SaveProvider(gomock.Any(), gomock.Any()).Return(assert.AnError)
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "select fails",
			args: args{
				ctx: context.Background(),
			},
			expectFn: func(mp *mock_auth.MockProvider, mm *Mockmanager) {
				mp.EXPECT().Test(gomock.Any()).Return(fixtures.LoadPtr[slack.AuthTestResponse](string(fixtures.TestAuthTestInfo)), nil)
				mm.EXPECT().SaveProvider(gomock.Any(), gomock.Any()).Return(nil)
				mm.EXPECT().Select(gomock.Any()).Return(assert.AnError)
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "success",
			args: args{
				ctx: context.Background(),
			},
			expectFn: func(mp *mock_auth.MockProvider, mm *Mockmanager) {
				mp.EXPECT().Test(gomock.Any()).Return(fixtures.LoadPtr[slack.AuthTestResponse](string(fixtures.TestAuthTestInfo)), nil)
				mm.EXPECT().SaveProvider(gomock.Any(), gomock.Any()).Return(nil)
				mm.EXPECT().Select(gomock.Any()).Return(nil)
			},
			want:    "test",
			wantErr: false,
		},
		{
			name: "url empty fails",
			args: args{
				ctx: context.Background(),
			},
			expectFn: func(mp *mock_auth.MockProvider, mm *Mockmanager) {
				mp.EXPECT().Test(gomock.Any()).Return(&slack.AuthTestResponse{URL: ""}, nil)
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "url sanitize fails",
			args: args{
				ctx: context.Background(),
			},
			expectFn: func(mp *mock_auth.MockProvider, mm *Mockmanager) {
				mp.EXPECT().Test(gomock.Any()).Return(&slack.AuthTestResponse{URL: "ftp://lol.example.com"}, nil)
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mp := mock_auth.NewMockProvider(ctrl)
			mm := NewMockmanager(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(mp, mm)
			}
			got, err := createAndSelect(tt.args.ctx, mm, mp)
			if (err != nil) != tt.wantErr {
				t.Errorf("createAndSelect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("createAndSelect() = %v, want %v", got, tt.want)
			}
		})
	}
}
