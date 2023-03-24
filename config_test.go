package slackdump

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLimits_Apply(t *testing.T) {
	type fields struct {
		Workers         int
		DownloadRetries int
		Tier2           TierLimits
		Tier3           TierLimits
		Request         RequestLimit
	}
	type args struct {
		other Limits
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    Limits
		wantErr bool
	}{
		{
			"change workers",
			fields{
				Workers: 3,
				Tier2:   TierLimits{Burst: 1},
				Tier3:   TierLimits{Burst: 1},
				Request: RequestLimit{
					Conversations: 50,
					Replies:       50,
					Channels:      50,
				},
			},
			args{Limits{Workers: 4}},
			Limits{
				Workers: 4,
				Tier2:   TierLimits{Burst: 1},
				Tier3:   TierLimits{Burst: 1},
				Request: RequestLimit{
					Conversations: 50,
					Replies:       50,
					Channels:      50,
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Limits{
				Workers:         tt.fields.Workers,
				DownloadRetries: tt.fields.DownloadRetries,
				Tier2:           tt.fields.Tier2,
				Tier3:           tt.fields.Tier3,
				Request:         tt.fields.Request,
			}
			if err := o.Apply(tt.args.other); (err != nil) != tt.wantErr {
				t.Errorf("o.Apply() error=%v wantErr=%v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestLimits_Validate(t *testing.T) {
	type fields struct {
		Workers         int
		DownloadRetries int
		Tier2           TierLimits
		Tier3           TierLimits
		Request         RequestLimit
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr assert.ErrorAssertionFunc
	}{
		{"validate default options",
			fields(DefLimits),
			func(t assert.TestingT, err error, i ...interface{}) bool {
				return err == nil
			},
		},
		{
			"empty options is an error",
			fields{},
			func(t assert.TestingT, err error, i ...interface{}) bool {
				if err == nil {
					t.Errorf("expected error, but got %v", err)
					return false
				}
				return true
			},
		},
		{
			"invalid workers",
			fields{
				Workers: -1,
				Tier2:   TierLimits{Burst: 1},
				Tier3:   TierLimits{Burst: 1},
				Request: RequestLimit{
					Conversations: 50,
					Replies:       50,
					Channels:      50,
				},
			},
			func(t assert.TestingT, err error, i ...interface{}) bool {
				if err == nil {
					t.Errorf("expected error, but got %v", err)
				}
				return err != nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Limits{
				Workers:         tt.fields.Workers,
				DownloadRetries: tt.fields.DownloadRetries,
				Tier2:           tt.fields.Tier2,
				Tier3:           tt.fields.Tier3,
				Request:         tt.fields.Request,
			}
			tt.wantErr(t, o.Validate(), "Validate()")
		})
	}
}
