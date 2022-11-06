package slackdump

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v2/logger"
)

func TestOptions_Apply(t *testing.T) {
	type fields struct {
		Workers             int
		DownloadRetries     int
		Tier2Boost          uint
		Tier2Burst          uint
		Tier2Retries        int
		Tier3Boost          uint
		Tier3Burst          uint
		Tier3Retries        int
		ConversationsPerReq int
		ChannelsPerReq      int
		RepliesPerReq       int
		DumpFiles           bool
		UserCacheFilename   string
		MaxUserCacheAge     time.Duration
		NoUserCache         bool
		CacheDir            string
		Logger              logger.Interface
	}
	type args struct {
		other Options
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *Options
		wantErr bool
	}{
		{
			"change workers",
			fields{
				Workers:             3,
				Tier2Burst:          1,
				Tier3Burst:          1,
				ConversationsPerReq: 50,
				RepliesPerReq:       50,
				ChannelsPerReq:      50,
			},
			args{Options{Workers: 4}},
			&Options{
				Workers:             4,
				Tier2Burst:          1,
				Tier3Burst:          1,
				ConversationsPerReq: 50,
				RepliesPerReq:       50,
				ChannelsPerReq:      50,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{
				Workers:             tt.fields.Workers,
				DownloadRetries:     tt.fields.DownloadRetries,
				Tier2Boost:          tt.fields.Tier2Boost,
				Tier2Burst:          tt.fields.Tier2Burst,
				Tier2Retries:        tt.fields.Tier2Retries,
				Tier3Boost:          tt.fields.Tier3Boost,
				Tier3Burst:          tt.fields.Tier3Burst,
				Tier3Retries:        tt.fields.Tier3Retries,
				ConversationsPerReq: tt.fields.ConversationsPerReq,
				ChannelsPerReq:      tt.fields.ChannelsPerReq,
				RepliesPerReq:       tt.fields.RepliesPerReq,
				DumpFiles:           tt.fields.DumpFiles,
				UserCacheFilename:   tt.fields.UserCacheFilename,
				MaxUserCacheAge:     tt.fields.MaxUserCacheAge,
				NoUserCache:         tt.fields.NoUserCache,
				CacheDir:            tt.fields.CacheDir,
				Logger:              tt.fields.Logger,
			}

			if err := o.Apply(tt.args.other); (err != nil) != tt.wantErr {
				t.Errorf("o.Apply() error=%v wantErr=%v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, o)
		})
	}
}

func TestOptions_Validate(t *testing.T) {
	type fields struct {
		Workers             int
		DownloadRetries     int
		Tier2Boost          uint
		Tier2Burst          uint
		Tier2Retries        int
		Tier3Boost          uint
		Tier3Burst          uint
		Tier3Retries        int
		ConversationsPerReq int
		ChannelsPerReq      int
		RepliesPerReq       int
		DumpFiles           bool
		UserCacheFilename   string
		MaxUserCacheAge     time.Duration
		NoUserCache         bool
		CacheDir            string
		Logger              logger.Interface
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr assert.ErrorAssertionFunc
	}{
		{"validate default options",
			fields(DefOptions),
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
				Workers:             -1,
				ChannelsPerReq:      10,
				RepliesPerReq:       10,
				ConversationsPerReq: 10,
				Tier3Burst:          1,
				Tier2Burst:          1,
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
			o := &Options{
				Workers:             tt.fields.Workers,
				DownloadRetries:     tt.fields.DownloadRetries,
				Tier2Boost:          tt.fields.Tier2Boost,
				Tier2Burst:          tt.fields.Tier2Burst,
				Tier2Retries:        tt.fields.Tier2Retries,
				Tier3Boost:          tt.fields.Tier3Boost,
				Tier3Burst:          tt.fields.Tier3Burst,
				Tier3Retries:        tt.fields.Tier3Retries,
				ConversationsPerReq: tt.fields.ConversationsPerReq,
				ChannelsPerReq:      tt.fields.ChannelsPerReq,
				RepliesPerReq:       tt.fields.RepliesPerReq,
				DumpFiles:           tt.fields.DumpFiles,
				UserCacheFilename:   tt.fields.UserCacheFilename,
				MaxUserCacheAge:     tt.fields.MaxUserCacheAge,
				NoUserCache:         tt.fields.NoUserCache,
				CacheDir:            tt.fields.CacheDir,
				Logger:              tt.fields.Logger,
			}
			tt.wantErr(t, o.Validate(), fmt.Sprintf("Validate()"))
		})
	}
}
