package resume

import (
	"context"
	_ "embed"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/internal/source/mock_source"
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
