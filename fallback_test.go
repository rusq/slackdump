package slackdump

import (
	"context"
	"errors"
	"testing"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/rusq/slackdump/v3/logger"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

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
