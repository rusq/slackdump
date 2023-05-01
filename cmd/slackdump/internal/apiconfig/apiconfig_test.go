package apiconfig

import (
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/rusq/slackdump/v2"
)

const (
	sampleLimitsYaml = `workers: 4
download_retries: 3
tier_2:
  boost: 20
  burst: 1
  retries: 20
tier_3:
  boost: 120
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
per_request:
  conversations: 100
  channels: 100
  replies: 200
`
)

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
			got, err := readLimits(tt.args.r)
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
