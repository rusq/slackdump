package slackdump

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/edge"
	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/rusq/slackdump/v3/logger"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_fallbackClient_getClient(t *testing.T) {
	type fields struct {
		cl        []clienter
		methodPtr map[fallbackMethod]int
		lg        logger.Interface
	}
	type args struct {
		m fallbackMethod
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    clienter
		wantErr bool
	}{
		{
			name: "returns available client",
			fields: fields{
				cl: []clienter{
					&slack.Client{},
				},
				methodPtr: map[fallbackMethod]int{},
				lg:        logger.Default,
			},
			args: args{
				m: fbAuthTestContext,
			},
			want:    &slack.Client{},
			wantErr: false,
		}, {
			name: "returns the next client",
			fields: fields{
				cl: []clienter{
					&slack.Client{},
					&edge.Wrapper{},
				},
				methodPtr: map[fallbackMethod]int{
					fbAuthTestContext: 1,
				},
				lg: logger.Default,
			},
			args: args{
				m: fbAuthTestContext,
			},
			want:    &edge.Wrapper{},
			wantErr: false,
		}, {
			name: "no clients",
			fields: fields{
				cl:        []clienter{},
				methodPtr: map[fallbackMethod]int{},
				lg:        logger.Default,
			},
			args: args{
				m: fbAuthTestContext,
			},
			want:    nil,
			wantErr: true,
		}, {
			name: "no next fallback client available",
			fields: fields{
				cl: []clienter{
					&slack.Client{},
				},
				methodPtr: map[fallbackMethod]int{
					fbAuthTestContext: 1,
				},
				lg: logger.Default,
			},
			args: args{
				m: fbAuthTestContext,
			},
			want:    nil,
			wantErr: true,
		}, {
			name: "calling the method that is not in the pointer map",
			fields: fields{
				cl: []clienter{
					&slack.Client{},
				},
				methodPtr: map[fallbackMethod]int{
					fbAuthTestContext: 0,
				},
				lg: logger.Default,
			},
			args: args{
				m: fbGetConversationHistoryContext,
			},
			want:    &slack.Client{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := &fallbackClient{
				cl:        tt.fields.cl,
				methodPtr: tt.fields.methodPtr,
				lg:        tt.fields.lg,
			}
			got, err := fc.getClient(tt.args.m)
			if (err != nil) != tt.wantErr {
				t.Errorf("fallbackClient.getClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fallbackClient.getClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fallbackClient_fallback(t *testing.T) {
	type fields struct {
		cl        []clienter
		methodPtr map[fallbackMethod]int
		lg        logger.Interface
	}
	type args struct {
		m fallbackMethod
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "fallbacks to the second client",
			fields: fields{
				cl: []clienter{
					&slack.Client{},
					&edge.Wrapper{},
				},
				methodPtr: map[fallbackMethod]int{
					fbAuthTestContext: 0,
				},
				lg: logger.Default,
			},
			args:    args{m: fbAuthTestContext},
			wantErr: false,
		}, {
			name: "nowhere to turn",
			fields: fields{
				cl: []clienter{
					&slack.Client{},
				},
				methodPtr: map[fallbackMethod]int{
					fbAuthTestContext: 0,
				},
				lg: logger.Default,
			},
			args:    args{m: fbAuthTestContext},
			wantErr: true,
		}, {
			name: "no clients",
			fields: fields{
				cl: []clienter{},
				methodPtr: map[fallbackMethod]int{
					fbAuthTestContext: 0,
				},
				lg: logger.Default,
			},
			args:    args{m: fbAuthTestContext},
			wantErr: true,
		}, {
			name: "no next fallback client available, and we're out of bounds (impossible, but why not)",
			fields: fields{
				cl: []clienter{
					&slack.Client{},
				},
				methodPtr: map[fallbackMethod]int{
					fbAuthTestContext: 1,
				},
				lg: logger.Default,
			},
			args:    args{m: fbAuthTestContext},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := &fallbackClient{
				cl:        tt.fields.cl,
				methodPtr: tt.fields.methodPtr,
				lg:        tt.fields.lg,
			}
			if err := fc.fallback(tt.args.m); (err != nil) != tt.wantErr {
				t.Errorf("fallbackClient.fallback() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_fallbackClient_GetConversationInfoContext(t *testing.T) {
	var (
		testParams = &slack.GetConversationInfoInput{ChannelID: "CSUCCESS"}
		ret        = fixtures.LoadPtr[slack.Channel](fixtures.TestChannel)
	)

	var tests = []struct {
		name    string
		expect  func(t *testing.T) []clienter
		want    *slack.Channel
		wantErr bool
	}{
		{
			name: "fallbacks",
			expect: func(t *testing.T) []clienter {
				ctrl := gomock.NewController(t)
				// main client, returning enterprise_is_restricted
				mcl := NewmockClienter(ctrl)
				mcl.EXPECT().GetConversationInfoContext(gomock.Any(), testParams).Return(nil, slack.SlackErrorResponse{Err: enterpriseIsRestricted})

				// fallback client, successfully executing.
				fmcl := NewmockClienter(ctrl)
				fmcl.EXPECT().GetConversationInfoContext(gomock.Any(), testParams).Return(ret, nil)

				return []clienter{mcl, fmcl}
			},
			want:    ret,
			wantErr: false,
		},
		{
			name: "no fallbacks",
			expect: func(t *testing.T) []clienter {
				ctrl := gomock.NewController(t)
				mcl := NewmockClienter(ctrl)
				mcl.EXPECT().GetConversationInfoContext(gomock.Any(), testParams).Return(nil, errors.New("error"))

				fcl := NewmockClienter(ctrl)
				return []clienter{mcl, fcl}
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clients := tt.expect(t)

			fc := &fallbackClient{
				cl:        clients,
				methodPtr: map[fallbackMethod]int{},
				lg:        logger.Default,
			}

			got, err := fc.GetConversationInfoContext(context.Background(), testParams)
			if (err != nil) != tt.wantErr {
				t.Errorf("fallbackClient.GetConversationInfoContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
