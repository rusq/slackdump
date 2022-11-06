package slackdump

// In this file: slackdump options.

import (
	"reflect"
	"runtime"
	"time"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"

	"github.com/rusq/slackdump/v2/logger"
)

const defNumWorkers = 4 // default number of file downloaders. it's here because it's used in several places.

// Options is the option set for the Session.
type Options struct {
	// number of file-saving workers
	Workers int `json:"workers,omitempty" yaml:"workers,omitempty" validate:"gte=1,lte=128"`
	// if we get rate limited on file downloads, this is how many times we're
	// going to retry
	DownloadRetries int `json:"download_retries,omitempty" yaml:"download_retries,omitempty"`
	// Tier-2 limiter boost
	Tier2Boost uint `json:"tier_2_boost,omitempty" yaml:"tier_2_boost,omitempty"`
	// Tier-2 limiter burst
	Tier2Burst uint `json:"tier_2_burst,omitempty" yaml:"tier_2_burst,omitempty" validate:"gte=1"`
	// Tier-2 retries when getting 429 on channels fetch
	Tier2Retries int `json:"tier_2_retries,omitempty" yaml:"tier_2_retries,omitempty"`
	// Tier-3 limiter boost allows to increase or decrease the slack Tier
	// req/min rate. Affects all tiers.
	Tier3Boost uint `json:"tier_3_boost,omitempty" yaml:"tier_3_boost,omitempty"`
	// Tier-3 limiter burst allows to set the limiter burst in req/sec. Default
	// of 1 is safe.
	Tier3Burst uint `json:"tier_3_burst,omitempty" yaml:"tier_3_burst,omitempty" validate:"gte=1"`
	// number of retries to do when getting 429 on conversation fetch
	Tier3Retries int `json:"tier_3_retries,omitempty" yaml:"tier_3_retries,omitempty"`
	// number of messages we get per 1 API request. bigger the number, fewer
	// requests, but they become more beefy.
	ConversationsPerReq int `json:"conversations_per_request,omitempty" yaml:"conversations_per_request,omitempty" validate:"gt=0,lte=100"`
	// number of channels to fetch per 1 API request.
	ChannelsPerReq int `json:"channels_per_request,omitempty" yaml:"channels_per_request,omitempty" validate:"gt=0,lte=1000"`
	// number of thread replies per request (slack default: 1000)
	RepliesPerReq int `json:"replies_per_request,omitempty" yaml:"replies_per_request,omitempty" validate:"gt=0,lte=1000"`

	// other parameters
	DumpFiles         bool             `json:"-" yaml:"-"` // will we save the conversation files?
	UserCacheFilename string           `json:"-" yaml:"-"` // user cache filename
	MaxUserCacheAge   time.Duration    `json:"-" yaml:"-"` // how long the user cache is valid for.
	NoUserCache       bool             `json:"-" yaml:"-"` // disable fetching users from the API.
	CacheDir          string           `json:"-" yaml:"-"` // cache directory
	Logger            logger.Interface `json:"-" yaml:"-"`
}

// DefOptions is the default options used when initialising slackdump instance.
var DefOptions = Options{
	Workers:             defNumWorkers, // number of workers doing the file download
	DownloadRetries:     3,             // this shouldn't even happen, as we have no limiter on files download.
	Tier2Boost:          20,            // seems to work fine with this boost
	Tier2Burst:          1,             // limiter will wait indefinitely if it is less than 1.
	Tier2Retries:        20,            // see #28, sometimes slack is being difficult
	Tier3Boost:          120,           // playing safe there, but generally value of 120 is fine.
	Tier3Burst:          1,             // safe value, who would ever want to modify it? I don't know.
	Tier3Retries:        3,             // on Tier 3 this was never a problem, even with limiter-boost=120
	ConversationsPerReq: 200,           // this is the recommended value by Slack. But who listens to them anyway.
	ChannelsPerReq:      200,           // channels are Tier2 rate limited. Slack is greedy and never returns more than 100 per call.
	RepliesPerReq:       200,           // the API-default is 1000 (see conversations.replies), but on large threads it may fail (see #54)

	DumpFiles:         false,
	UserCacheFilename: "users.cache", // seems logical
	MaxUserCacheAge:   4 * time.Hour, // quick math:  that's 1/6th of a day, how's that, huh?
	CacheDir:          "",            // default cache dir
	Logger:            logger.Default,
}

var (
	optValidator       *validator.Validate
	OptErrTranslations ut.Translator
)

func init() {
	optValidator = validator.New()
	english := en.New()
	uni := ut.New(english, english)
	var ok bool
	OptErrTranslations, ok = uni.GetTranslator("en")
	if !ok {
		panic("internal error: failed to init translator")
	}
	if err := en_translations.RegisterDefaultTranslations(optValidator, OptErrTranslations); err != nil {
		panic(err)
	}
}

// Apply applies changes from other Options. It affects only API-limits related
// values.
func (o *Options) Apply(other Options) error {
	apply(&o.Workers, other.Workers)
	apply(&o.DownloadRetries, other.DownloadRetries)
	apply(&o.Tier2Boost, other.Tier2Boost)
	apply(&o.Tier2Burst, other.Tier2Burst)
	apply(&o.Tier2Retries, other.Tier2Retries)
	apply(&o.Tier3Boost, other.Tier3Boost)
	apply(&o.Tier3Burst, other.Tier3Burst)
	apply(&o.Tier3Retries, other.Tier3Retries)
	apply(&o.ConversationsPerReq, other.ConversationsPerReq)
	apply(&o.ChannelsPerReq, other.ChannelsPerReq)
	apply(&o.RepliesPerReq, other.RepliesPerReq)
	return o.Validate()
}

func (o *Options) Validate() error {
	return optValidator.Struct(o)
}

func apply[T comparable](this *T, other T) {
	if !isZero(other) && !(*this == other) {
		*this = other
	}
}

func isZero(a any) bool {
	return a == reflect.Zero(reflect.TypeOf(a)).Interface()
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
// base slack Tier limits.  The resulting
// events per minute will be calculated like this:
//
//	events_per_sec =  (<slack_tier_epm> + <eventsPerMin>) / 60.0
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
// base slack Tier limits.  The resulting
// events per minute will be calculated like this:
//
//	events_per_sec =  (<slack_tier_epm> + <eventsPerMin>) / 60.0
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

// WithLogger allows to set the custom logger.
func WithLogger(l logger.Interface) Option {
	return func(o *Options) {
		if l == nil {
			l = logger.Default
		}
		o.Logger = l
	}
}

func CacheDir(dir string) Option {
	return func(o *Options) {
		if dir == "" {
			return
		}
		o.CacheDir = dir
	}
}
