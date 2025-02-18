package repository

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
)

func Test_workspaceRepository_GetWorkspace(t *testing.T) {
	var (
		wsp1, _ = NewDBWorkspace(1, &slack.AuthTestResponse{
			URL:    "http://lol.slack.com",
			Team:   "lolzteam",
			User:   "lolzuser",
			TeamID: "T123456",
			UserID: "U123456",
		})
		wsp2, _ = NewDBWorkspace(2, &slack.AuthTestResponse{
			URL:    wsp1.URL,
			Team:   wsp1.Team,
			User:   "renameduser",
			TeamID: wsp1.TeamID,
			UserID: wsp1.UserID,
		})
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
				ctx:  context.Background(),
				conn: testConn(t),
			},
			prepFn: func(t *testing.T, conn PrepareExtContext) {
				t.Helper()
				prepChunk(chunk.CWorkspaceInfo, chunk.CWorkspaceInfo)(t, conn)
				wr := NewWorkspaceRepository()
				if err := wr.Insert(context.Background(), conn, wsp1, wsp2); err != nil {
					t.Fatal(err)
				}
			},
			want: *wsp2,
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
