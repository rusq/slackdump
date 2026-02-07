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
package slackdump

import (
	"context"
	"log"
	"math"
	"os"
	"testing"
	"time"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/internal/client/mock_client"
	"github.com/rusq/slackdump/v3/internal/network"
)

func Test_newLimiter(t *testing.T) {
	t.Parallel()
	type args struct {
		t     network.Tier
		burst uint
		boost int
	}
	tests := []struct {
		name      string
		args      args
		wantDelay time.Duration
	}{
		{
			"Tier test",
			args{
				network.Tier3,
				1,
				0,
			},
			time.Duration(math.Round(60.0/float64(network.Tier3)*1000.0)) * time.Millisecond, // 6/5 sec
		},
		{
			"burst 2",
			args{
				network.Tier3,
				2,
				0,
			},
			1 * time.Millisecond,
		},
		{
			"boost 70",
			args{
				network.Tier3,
				1,
				70,
			},
			time.Duration(math.Round(60.0/float64(network.Tier3+70)*1000.0)) * time.Millisecond, // 500 msec
		},
	}
	for _, test := range tests {
		tt := test
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := network.NewLimiter(tt.args.t, tt.args.burst, tt.args.boost)

			assert.NoError(t, got.Wait(t.Context())) // prime

			start := time.Now()
			err := got.Wait(t.Context())
			stop := time.Now()

			assert.NoError(t, err)
			assert.WithinDurationf(t, start.Add(tt.wantDelay), stop, 15*time.Millisecond, "delayed for: %s, expected: %s", stop.Sub(start), tt.wantDelay)
		})
	}
}

func ExampleNew_tokenAndCookie() {
	provider, err := auth.NewValueAuth("xoxc-...", "xoxd-...")
	if err != nil {
		log.Print(err)
		return
	}
	fsa := openTempFS()
	defer fsa.Close()

	sd, err := New(context.Background(), provider)
	if err != nil {
		log.Print(err)
		return
	}
	_ = sd
}

func ExampleNew_cookieFile() {
	provider, err := auth.NewCookieFileAuth("xoxc-...", "cookies.txt")
	if err != nil {
		log.Print(err)
		return
	}
	fsa := openTempFS()
	defer fsa.Close()

	sd, err := New(context.Background(), provider)
	if err != nil {
		log.Print(err)
		return
	}
	_ = sd
}

func ExampleNew_browserAuth() {
	provider, err := auth.NewPlaywrightAuth(context.Background())
	if err != nil {
		log.Print(err)
		return
	}
	fsa := openTempFS()
	defer fsa.Close()
	sd, err := New(context.Background(), provider)
	if err != nil {
		log.Print(err)
		return
	}
	_ = sd
}

func openTempFS() fsadapter.FSCloser {
	dir, err := os.MkdirTemp("", "slackdump")
	if err != nil {
		panic(err)
	}
	fsc, err := fsadapter.New(dir)
	if err != nil {
		panic(err)
	}
	return fsc
}

func TestSession_initWorkspaceInfo(t *testing.T) {
	ctx := t.Context()
	t.Run("ok", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mc := mock_client.NewMockSlackClienter(ctrl)
		mc.EXPECT().AuthTestContext(gomock.Any()).Return(&slack.AuthTestResponse{
			TeamID: "TEST",
		}, nil)
		s := Session{
			client: nil, // it should use the provided client
		}

		err := s.initWorkspaceInfo(ctx, mc)
		assert.NoError(t, err, "unexpected initialisation error")
	})
	t.Run("error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mc := mock_client.NewMockSlackClienter(ctrl)
		mc.EXPECT().AuthTestContext(gomock.Any()).Return(nil, assert.AnError)
		s := Session{
			client: nil, // it should use the provided client
		}
		err := s.initWorkspaceInfo(ctx, mc)
		assert.Error(t, err, "expected error")
	})
}
