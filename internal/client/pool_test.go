package client

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/rusq/slack"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v3/internal/client/mock_client"
)

func TestPool_next(t *testing.T) {
	type fields struct {
		pool     []Slack
		strategy strategy
	}
	tests := []struct {
		name      string
		fields    fields
		want      Slack
		wantPanic bool
	}{
		{
			name: "empty pool",
			fields: fields{
				pool:     []Slack{},
				strategy: newRoundRobin(0),
			},
			want:      nil,
			wantPanic: true,
		},
		{
			name: "single client",
			fields: fields{
				pool:     []Slack{&mock_client.MockSlack{}},
				strategy: newRoundRobin(1),
			},
			want:      &mock_client.MockSlack{},
			wantPanic: false,
		},
		{
			name: "multiple clients",
			fields: fields{
				pool:     []Slack{&mock_client.MockSlack{}, &mock_client.MockSlack{}},
				strategy: newRoundRobin(2),
			},
			want:      &mock_client.MockSlack{},
			wantPanic: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); (r != nil) != tt.wantPanic {
					t.Errorf("Pool.next() panic = %v, wantPanic %v", r, tt.wantPanic)
				}
			}()
			p := &Pool{
				pool:     tt.fields.pool,
				strategy: tt.fields.strategy,
			}
			if got := p.next(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Pool.next() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPool_AuthTestContext(t *testing.T) {
	type fields struct {
		// pool     []Slack
		strategy strategy
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		numClients   int
		expectFn     func([]*mock_client.MockSlack)
		wantResponse *slack.AuthTestResponse
		wantErr      bool
	}{
		{
			name: "expect call on the second client (round robin)",
			fields: fields{
				strategy: newRoundRobin(2),
			},
			args: args{
				ctx: t.Context(),
			},
			numClients: 2,
			expectFn: func(clients []*mock_client.MockSlack) {
				clients[0].EXPECT().AuthTestContext(t.Context()).Times(0)
				clients[1].EXPECT().AuthTestContext(t.Context()).Return(&slack.AuthTestResponse{URL: "abc"}, nil)
			},
			wantResponse: &slack.AuthTestResponse{URL: "abc"},
			wantErr:      false,
		},
		{
			name: "expect call on the first client (round robin)",
			fields: fields{
				strategy: &roundRobin{total: 2, i: 1},
			},
			args: args{
				ctx: t.Context(),
			},
			numClients: 2,
			expectFn: func(clients []*mock_client.MockSlack) {
				clients[0].EXPECT().AuthTestContext(t.Context()).Return(&slack.AuthTestResponse{URL: "abc"}, nil)
				clients[1].EXPECT().AuthTestContext(t.Context()).Times(0)
			},
			wantResponse: &slack.AuthTestResponse{URL: "abc"},
			wantErr:      false,
		},
		{
			name: "expect call on the first client (round robin) with 1 client",
			fields: fields{
				strategy: newRoundRobin(1),
			},
			args: args{
				ctx: t.Context(),
			},
			numClients: 1,
			expectFn: func(clients []*mock_client.MockSlack) {
				clients[0].EXPECT().AuthTestContext(t.Context()).Return(&slack.AuthTestResponse{URL: "abc"}, nil)
			},
			wantResponse: &slack.AuthTestResponse{URL: "abc"},
			wantErr:      false,
		},
		{
			name: "expect call on the first client (round robin) with 1 client and error",
			fields: fields{
				strategy: newRoundRobin(1),
			},
			args: args{
				ctx: t.Context(),
			},
			numClients: 1,
			expectFn: func(clients []*mock_client.MockSlack) {
				clients[0].EXPECT().AuthTestContext(t.Context()).Return(nil, errors.New("error"))
			},
			wantResponse: nil,
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mcs := make([]*mock_client.MockSlack, tt.numClients)
			pool := make([]Slack, len(mcs))
			for i := 0; i < tt.numClients; i++ {
				mcs[i] = mock_client.NewMockSlack(ctrl)
				pool[i] = mcs[i]
			}
			if tt.expectFn != nil {
				tt.expectFn(mcs)
			}
			p := &Pool{
				pool:     pool,
				strategy: tt.fields.strategy,
			}
			gotResponse, err := p.AuthTestContext(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Pool.AuthTestContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResponse, tt.wantResponse) {
				t.Errorf("Pool.AuthTestContext() = %v, want %v", gotResponse, tt.wantResponse)
			}
		})
	}
}
