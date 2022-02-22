package slackdump

// In this file: slackdump options.

import (
	"runtime"
	"time"
)

const defNumWorkers = 4 // default number of file downloaders. it's here because it's used in several places.

// Options is the option set for the slackdumper.
type Options struct {
	DumpFiles           bool          // will we save the conversation files?
	Workers             int           // number of file-saving workers
	DownloadRetries     int           // if we get rate limited on file downloads, this is how many times we're going to retry
	Tier2Boost          uint          // tier-2 limiter boost
	Tier2Burst          uint          // tier-2 limiter burst
	Tier2Retries        int           // tier-2 retries when getting 429 on channels fetch
	Tier3Boost          uint          // tier-3 limiter boost allows to increase or decrease the slack tier req/min rate.  Affects all tiers.
	Tier3Burst          uint          // tier-3 limiter burst allows to set the limiter burst in req/sec.  Default of 1 is safe.
	Tier3Retries        int           // number of retries to do when getting 429 on conversation fetch
	ConversationsPerReq int           // number of messages we get per 1 api request. bigger the number, less requests, but they become more beefy.
	UserCacheFilename   string        // user cache filename
	MaxUserCacheAge     time.Duration // how long the user cache is valid for.
}

// DefOptions is the default options used when initialising slackdump instance.
var DefOptions = Options{
	DumpFiles:           false,
	Workers:             defNumWorkers, // number of workers doing the file download
	DownloadRetries:     3,             // this shouldn't even happen, as we have no limiter on files download.
	Tier2Boost:          0,             // slack is being difficult, so no boost for tier 2.
	Tier2Burst:          1,             // limiter will wait indefinitely if it is less than 1.
	Tier2Retries:        20,            // see #28, sometimes slack is being difficult
	Tier3Boost:          120,           // playing safe there, but generally value of 120 is fine.
	Tier3Burst:          1,             // safe value, who would ever want to modify it? I don't know.
	Tier3Retries:        3,             // on tier 3 this was never a problem, even with limiter-boost=120
	ConversationsPerReq: 200,           // this is the recommended value by Slack. But who listens to them anyway.
	UserCacheFilename:   "users.json",  // seems logical
	MaxUserCacheAge:     4 * time.Hour, // quick math:  that's 1/6th of a day, how's that, huh?
}

// Option is the signature of the option-setting function.
type Option func(*Options)

// DownloadFiles enables or disables the conversation/thread file downloads.
func DownloadFiles(b bool) Option {
	return func(options *Options) {
		options.DumpFiles = b
	}
}

// RetryThreads sets the number of attempts when dumping conversations and
// threads, and getting rate limited.
func RetryThreads(attempts int) Option {
	return func(options *Options) {
		if attempts > 0 {
			options.Tier3Retries = attempts
		}
	}
}

// RetryDownloads sets the number of attempts to download a file when getting
// rate limited.
func RetryDownloads(attempts int) Option {
	return func(options *Options) {
		if attempts > 0 {
			options.DownloadRetries = attempts
		}
	}
}

// Tier3Boost allows to deliver a magic kick to the limiter, to override the
// base slack tier limits.  The resulting
// events per minute will be calculated like this:
//
//   events_per_sec =  (<slack_tier_epm> + <eventsPerMin>) / 60.0
func Tier3Boost(eventsPerMin uint) Option {
	return func(options *Options) {
		options.Tier3Boost = eventsPerMin
	}
}

// Tier3Burst allows to set the limiter burst value.
func Tier3Burst(eventsPerSec uint) Option {
	return func(options *Options) {
		options.Tier3Burst = eventsPerSec
	}
}

// Tier2Boost allows to deliver a magic kick to the limiter, to override the
// base slack tier limits.  The resulting
// events per minute will be calculated like this:
//
//   events_per_sec =  (<slack_tier_epm> + <eventsPerMin>) / 60.0
func Tier2Boost(eventsPerMin uint) Option {
	return func(options *Options) {
		options.Tier2Boost = eventsPerMin
	}
}

// Tier2Burst allows to set the limiter burst value.
func Tier2Burst(eventsPerSec uint) Option {
	return func(options *Options) {
		options.Tier2Burst = eventsPerSec
	}
}

// NumWorkers allows to set the number of file download workers. n should be in
// range [1, NumCPU]. If not in range, will be reset to a defNumWorkers number,
// which seems reasonable.
func NumWorkers(n int) Option {
	return func(options *Options) {
		if n < 1 || runtime.NumCPU() < n {
			n = defNumWorkers
		}
		options.Workers = n
	}
}

// UserCacheFilename allows to set the user cache filename.
func UserCacheFilename(s string) Option {
	return func(options *Options) {
		if s != "" {
			options.UserCacheFilename = s
		}
	}
}

// MaxUserCacheAge allows to set the maximum user cache age.  If set to 0 - it
// will always use the API output, and never load cache.
func MaxUserCacheAge(d time.Duration) Option {
	return func(options *Options) {
		options.MaxUserCacheAge = d
	}
}
