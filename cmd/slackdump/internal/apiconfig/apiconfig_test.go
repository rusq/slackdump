package apiconfig

import (
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/rusq/slackdump/v2"
)

const sampleLimitsYaml = `workers: 4
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
			slackdump.DefOptions.Limits,
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
			args{strings.NewReader("workers: 55")},
			slackdump.Limits{
				Workers: 55,
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
