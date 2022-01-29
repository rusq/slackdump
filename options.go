package slackdump

import (
	"runtime"
	"time"
)

const defNumWorkers = 4 // default number of file downloaders. it's here because it's used in several places.

// options is the option set for the slackdumper.
type options struct {
	dumpfiles               bool          // will we save the conversation files?
	downloadWorkers         int           // number of file-saving workers
	conversationRetries     int           // number of retries to do when getting 429 on conversation fetch
	downloadRetries         int           // if we get rate limited on file downloads, this is how many times we're going to retry
	conversationsPerRequest int           // number of messages we get per 1 api request. bigger the number, less requests, but they become more beefy.
	limiterBoost            uint          // limiter boost allows to increase or decrease the slack tier req/min rate.  Affects all tiers.
	limiterBurst            uint          // limiter burst allows to set the limiter burst in req/sec.  Default of 1 is safe.
	userCacheFilename       string        // user cache filename
	maxUserCacheAge         time.Duration // how long the user cache is valid for.
}

// defOptions is the default options used when initialising slackdump instance.
var defOptions = options{
	downloadWorkers:         defNumWorkers, // number of workers doing the file download
	conversationRetries:     3,
	downloadRetries:         3,
	limiterBoost:            0,             // playing safe there, but generally value of 120 is fine.
	limiterBurst:            1,             // safe value, who would ever want to modify it? I don't know.
	conversationsPerRequest: 200,           // this is the recommended value by Slack. But who listens to them anyway.
	userCacheFilename:       "users.json",  // seems logical
	maxUserCacheAge:         4 * time.Hour, // quick math: that's 1/6th of a day.
}

// Option is the signature of the option-setting function.
type Option func(*SlackDumper)

// DownloadFiles enables or disables the conversation/thread file downloads.
func DownloadFiles(b bool) Option {
	return func(sd *SlackDumper) {
		sd.options.dumpfiles = b
	}
}

// RetryThreads sets the number of attempts when dumping conversations and
// threads, and getting rate limited.
func RetryThreads(attempts int) Option {
	return func(sd *SlackDumper) {
		if attempts > 0 {
			sd.options.conversationRetries = attempts
		}
	}
}

// RetryDownloads sets the number of attempts to download a file when getting
// rate limited.
func RetryDownloads(attempts int) Option {
	return func(sd *SlackDumper) {
		if attempts > 0 {
			sd.options.downloadRetries = attempts
		}
	}
}

// LimiterBoost allows to deliver a magic kick to the limiter, to override the
// base slack tier limits.  The resulting
// events per minute will be calculated like this:
//
//   events_per_sec =  (<slack_tier_epm> + <eventsPerMin>) / 60.0
func LimiterBoost(eventsPerMin uint) Option {
	return func(sd *SlackDumper) {
		sd.options.limiterBoost = eventsPerMin
	}
}

// LimiterBurst allows to set the limiter burst value.
func LimiterBurst(eventsPerSec uint) Option {
	return func(sd *SlackDumper) {
		sd.options.limiterBurst = eventsPerSec
	}
}

// NumWorkers allows to set the number of file download workers. n should be in
// range [1, NumCPU]. If not in range, will be reset to a defNumWorkers number,
// which seems reasonable.
func NumWorkers(n int) Option {
	return func(sd *SlackDumper) {
		if n < 1 || runtime.NumCPU() < n {
			n = defNumWorkers
		}
		sd.options.downloadWorkers = n
	}
}

// UserCacheFilename allows to set the user cache filename.
func UserCacheFilename(s string) Option {
	return func(sd *SlackDumper) {
		if s != "" {
			sd.options.userCacheFilename = s
		}
	}
}

// MaxUserCacheAge allows to set the maximum user cache age.  If set to 0 - it
// will always use the API output, and never load cache.
func MaxUserCacheAge(d time.Duration) Option {
	return func(sd *SlackDumper) {
		sd.options.maxUserCacheAge = d
	}
}
