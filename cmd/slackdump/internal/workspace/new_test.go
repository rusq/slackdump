package workspace

import (
	"context"
	"errors"
	"testing"

	auth "github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/logger"
	"go.uber.org/mock/gomock"
)

func init() {
	cfg.Log = logger.Silent
}

func Test_createWsp(t *testing.T) {
	type args struct {
		ctx       context.Context
		wsp       string
		confirmed bool
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(*Mockmanager)
		wantErr  bool
	}{
		{
			name: "success", // I
			args: args{
				ctx:       context.Background(),
				wsp:       "test",
				confirmed: false,
			},
			expectFn: func(m *Mockmanager) {
				m.EXPECT().Exists("test").Return(false)
				m.EXPECT().Auth(gomock.Any(), "test", gomock.Any()).Return(nil, nil)
				m.EXPECT().Select("test").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "exist, ask- no", // VIII, II
			args: args{
				ctx:       context.Background(),
				wsp:       "test",
				confirmed: false,
			},
			expectFn: func(m *Mockmanager) {
				m.EXPECT().Exists("test").Return(true)
				canOverwrite = func(string) bool {
					// decline overwrite
					return false
				}
			},
			wantErr: true,
		},
		{
			name: "exist, skip interactive confirmation, but delete fails",
			args: args{
				ctx:       context.Background(),
				wsp:       "test",
				confirmed: true,
			},
			expectFn: func(m *Mockmanager) {
				m.EXPECT().Exists("test").Return(true)
				m.EXPECT().Delete("test").Return(errors.New("fail"))
			},
			wantErr: true,
		},
		{
			name: "exist, ask- yes, delete fails", // VIII, III
			args: args{
				ctx:       context.Background(),
				wsp:       "test",
				confirmed: false, // so will ask
			},
			expectFn: func(m *Mockmanager) {
				m.EXPECT().Exists("test").Return(true)
				canOverwrite = func(string) bool {
					// confirm overwrite
					return true
				}
				m.EXPECT().Delete("test").Return(errors.New("fail"))
			},
			wantErr: true,
		},
		{
			name: "auth fails", // IV, V
			args: args{
				ctx:       context.Background(),
				wsp:       "test",
				confirmed: false,
			},
			expectFn: func(m *Mockmanager) {
				m.EXPECT().Exists("test").Return(false)
				m.EXPECT().Auth(gomock.Any(), "test", gomock.Any()).Return(nil, errors.New("fail"))
			},
			wantErr: true,
		},
		{
			name: "auth cancelled", // IV, IX
			args: args{
				ctx:       context.Background(),
				wsp:       "test",
				confirmed: false,
			},
			expectFn: func(m *Mockmanager) {
				m.EXPECT().Exists("test").Return(false)
				m.EXPECT().Auth(gomock.Any(), "test", gomock.Any()).Return(nil, auth.ErrCancelled)
			},
			wantErr: true,
		},
		{
			name: "select fails", // I -> VII
			args: args{
				ctx:       context.Background(),
				wsp:       "test",
				confirmed: false,
			},
			expectFn: func(m *Mockmanager) {
				m.EXPECT().Exists("test").Return(false)
				m.EXPECT().Auth(gomock.Any(), "test", gomock.Any()).Return(nil, nil)
				m.EXPECT().Select("test").Return(errors.New("fail"))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := NewMockmanager(ctrl)
			tt.expectFn(m)
			if err := createWsp(tt.args.ctx, m, tt.args.wsp, tt.args.confirmed); (err != nil) != tt.wantErr {
				t.Errorf("createWsp() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
