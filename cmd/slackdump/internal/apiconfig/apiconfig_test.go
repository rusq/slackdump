package apiconfig

import (
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/rusq/slackdump/v3"
	"github.com/stretchr/testify/assert"
)

const (
	sampleLimitsYaml = `# yaml-language-server: $schema=https://raw.githubusercontent.com/rusq/slackdump/cli-remake/cmd/slackdump/internal/apiconfig/schema.json
workers: 4
download_retries: 3
tier_2:
    boost: 20
    burst: 1
    retries: 20
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

var testLimits = slackdump.Limits{
	Workers:         4,
	DownloadRetries: 3,
	Tier2: slackdump.TierLimit{
		Boost:   20,
		Burst:   1,
		Retries: 20,
	},
	Tier3: slackdump.TierLimit{
		Boost:   120,
		Burst:   1,
		Retries: 3,
	},
	Tier4: slackdump.TierLimit{
		Boost:   10,
		Burst:   1,
		Retries: 3,
	},
	Request: slackdump.RequestLimit{
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
		want    slackdump.Limits
		wantErr bool
	}{
		{
			"sample config (ok)",
			args{strings.NewReader(sampleLimitsYaml)},
			slackdump.DefLimits,
			false,
		},
		{
			"workers invalid",
			args{strings.NewReader("workers: -1")},
			slackdump.Limits{},
			true,
		},
		{
			"one parameter override",
			args{strings.NewReader(updatedConfigYaml)},
			slackdump.Limits{
				Workers:         55,
				DownloadRetries: 3,
				Tier2: slackdump.TierLimit{
					Boost:   20,
					Burst:   1,
					Retries: 330,
				},
				Tier3: slackdump.TierLimit{
					Boost:   120,
					Burst:   1,
					Retries: 3,
				},
				Tier4: slackdump.TierLimit{
					Boost:   10,
					Burst:   1,
					Retries: 3,
				},
				Request: slackdump.RequestLimit{
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
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_writeLimits(t *testing.T) {
	type args struct {
		cfg slackdump.Limits
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
