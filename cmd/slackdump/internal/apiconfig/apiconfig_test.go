package apiconfig

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/rusq/slackdump/v3/internal/network"
	"github.com/stretchr/testify/assert"
)

const (
	sampleLimitsYaml = `# yaml-language-server: $schema=https://raw.githubusercontent.com/rusq/slackdump/v3/cmd/slackdump/internal/apiconfig/schema.json
workers: 4
download_retries: 3
tier_2:
    boost: 20
    burst: 3
    retries: 20
tier_3:
    boost: 120
    burst: 5
    retries: 3
tier_4:
    boost: 10
    burst: 7
    retries: 3
per_request:
    conversations: 100
    channels: 100
    replies: 200
`
	// workers set to 55 in this one, tier2.retries to 330
	updatedConfigYaml = `workers: 55
download_retries: 3
tier_2:
  boost: 20
  burst: 1
  retries: 330
tier_3:
  boost: 120
  burst: 1
  retries: 3
tier_4:
  boost: 10
  burst: 1
  retries: 3
per_request:
  conversations: 100
  channels: 100
  replies: 200
`
)

var testLimits = network.Limits{
	Workers:         4,
	DownloadRetries: 3,
	Tier2: network.TierLimit{
		Boost:   20,
		Burst:   3,
		Retries: 20,
	},
	Tier3: network.TierLimit{
		Boost:   120,
		Burst:   5,
		Retries: 3,
	},
	Tier4: network.TierLimit{
		Boost:   10,
		Burst:   7,
		Retries: 3,
	},
	Request: network.RequestLimit{
		Conversations: 100,
		Channels:      100,
		Replies:       200,
	},
}

func Test_readConfig(t *testing.T) {
	type args struct {
		r io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    network.Limits
		wantErr bool
	}{
		{
			"sample config (ok)",
			args{strings.NewReader(sampleLimitsYaml)},
			network.DefLimits,
			false,
		},
		{
			"workers invalid",
			args{strings.NewReader("workers: -1")},
			network.Limits{},
			true,
		},
		{
			"one parameter override",
			args{strings.NewReader(updatedConfigYaml)},
			network.Limits{
				Workers:         55,
				DownloadRetries: 3,
				Tier2: network.TierLimit{
					Boost:   20,
					Burst:   1,
					Retries: 330,
				},
				Tier3: network.TierLimit{
					Boost:   120,
					Burst:   1,
					Retries: 3,
				},
				Tier4: network.TierLimit{
					Boost:   10,
					Burst:   1,
					Retries: 3,
				},
				Request: network.RequestLimit{
					Channels:      100,
					Conversations: 100,
					Replies:       200,
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := applyLimits(tt.args.r)
			t.Log(tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("readConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_writeLimits(t *testing.T) {
	type args struct {
		cfg network.Limits
	}
	tests := []struct {
		name    string
		args    args
		wantW   string
		wantErr bool
	}{
		{
			"writes limits and comments",
			args{testLimits},
			sampleLimitsYaml,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := writeLimits(w, tt.args.cfg); (err != nil) != tt.wantErr {
				t.Errorf("writeLimits() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.wantW, w.String())
		})
	}
}
