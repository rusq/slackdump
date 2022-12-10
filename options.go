package slackdump

// In this file: slackdump options.

import (
	"reflect"
	"runtime"
	"time"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	translations "github.com/go-playground/validator/v10/translations/en"

	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/rusq/slackdump/v2/logger"
)

const defNumWorkers = 4 // default number of file downloaders. it's here because it's used in several places.

// Options is the option set for the Session.
type Options struct {
	Limits Limits

	DumpFiles         bool          // will we save the conversation files?
	UserCacheFilename string        // user cache filename
	MaxUserCacheAge   time.Duration // how long the user cache is valid for.
	NoUserCache       bool          // disable fetching users from the API.
	CacheDir          string        // cache directory
	Logger            logger.Interface
	Filesystem        fsadapter.FS
}

type Limits struct {
	// number of file-saving workers
	Workers int `json:"workers,omitempty" yaml:"workers,omitempty" validate:"gte=1,lte=128"`
	// if we get rate limited on file downloads, this is how many times we're
	// going to retry
	DownloadRetries int `json:"download_retries,omitempty" yaml:"download_retries,omitempty"`
	// Tier-2 limits
	Tier2 TierLimits `json:"tier_2,omitempty" yaml:"tier_2,omitempty"`
	// Tier-3 limits
	Tier3 TierLimits `json:"tier_3,omitempty" yaml:"tier_3,omitempty"`
	// Request Limits
	Request RequestLimit `json:"per_request,omitempty" yaml:"per_request,omitempty"`
}

// TierLimits represents a Slack API Tier limits.
type TierLimits struct {
	// Tier limiter boost
	Boost uint `json:"boost,omitempty" yaml:"boost,omitempty"`
	// Tier limiter burst
	Burst uint `json:"burst,omitempty" yaml:"burst,omitempty" validate:"gte=1"`
	// Tier retries when getting 429 on channels fetch
	Retries int `json:"retries,omitempty" yaml:"retries,omitempty"`
}

// RequestLimit defines the limits on the requests that are sent to the API.
type RequestLimit struct {
	// number of messages we get per 1 API request. bigger the number, fewer
	// requests, but they become more beefy.
	Conversations int `json:"conversations,omitempty" yaml:"conversations,omitempty" validate:"gt=0,lte=100"`
	// number of channels to fetch per 1 API request.
	Channels int `json:"channels,omitempty" yaml:"channels,omitempty" validate:"gt=0,lte=1000"`
	// number of thread replies per request (slack default: 1000)
	Replies int `json:"replies,omitempty" yaml:"replies,omitempty" validate:"gt=0,lte=1000"`
}

// DefOptions is the default options used when initialising slackdump instance.
var DefOptions = Options{
	Limits: Limits{
		Workers:         defNumWorkers, // number of workers doing the file download
		DownloadRetries: 3,             // this shouldn't even happen, as we have no limiter on files download.
		Tier2: TierLimits{
			Boost:   20, // seems to work fine with this boost
			Burst:   1,  // limiter will wait indefinitely if it is less than 1.
			Retries: 20, // see #28, sometimes slack is being difficult
		},
		Tier3: TierLimits{
			Boost:   120, // playing safe there, but generally value of 120 is fine.
			Burst:   1,   // safe value, who would ever want to modify it? I don't know.
			Retries: 3,   // on Tier 3 this was never a problem, even with limiter-boost=120
		},
		Request: RequestLimit{
			Conversations: 100, // this is the recommended value by Slack. But who listens to them anyway.
			Channels:      100, // channels are Tier2 rate limited. Slack is greedy and never returns more than 100 per call.
			Replies:       200, // the API-default is 1000 (see conversations.replies), but on large threads it may fail (see #54)
		},
	},
	DumpFiles:         false,
	UserCacheFilename: "users.cache",               // seems logical
	MaxUserCacheAge:   4 * time.Hour,               // quick math:  that's 1/6th of a day, how's that, huh?
	CacheDir:          "",                          // default cache dir
	Logger:            logger.Default,              // default logger is the... default logger
	Filesystem:        fsadapter.NewDirectory("."), // default filesystem is the current directory
}

var (
	optValidator *validator.Validate // options validator
	// OptErrTranslations is the english translations for the validation
	// errors.
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
	if err := translations.RegisterDefaultTranslations(optValidator, OptErrTranslations); err != nil {
		panic(err)
	}
}

// Apply applies differing values from other Options. It only affects API-limits
// related values.
func (o *Limits) Apply(other Limits) error {
	apply(&o.Workers, other.Workers)
	apply(&o.DownloadRetries, other.DownloadRetries)
	apply(&o.Tier2.Boost, other.Tier2.Boost)
	apply(&o.Tier2.Burst, other.Tier2.Burst)
	apply(&o.Tier2.Retries, other.Tier2.Retries)
	apply(&o.Tier3.Boost, other.Tier3.Boost)
	apply(&o.Tier3.Burst, other.Tier3.Burst)
	apply(&o.Tier3.Retries, other.Tier3.Retries)
	apply(&o.Request.Conversations, other.Request.Conversations)
	apply(&o.Request.Channels, other.Request.Channels)
	apply(&o.Request.Replies, other.Request.Replies)
	return o.Validate()
}

func (o *Limits) Validate() error {
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

// RetryDownloads sets the number of attempts to download a file when getting
// rate limited.
func RetryDownloads(attempts int) Option {
	return func(options *Options) {
		if attempts > 0 {
			options.Limits.DownloadRetries = attempts
		}
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
		options.Limits.Workers = n
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

// WithLimits applies the Limits to the default options.
func WithLimits(l Limits) Option {
	return func(o *Options) {
		_ = o.Limits.Apply(l) // NewWithOptions runs the validation.
	}
}
