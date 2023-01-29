package slackdump

// In this file: slackdump options.

import (
	"reflect"
	"time"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	translations "github.com/go-playground/validator/v10/translations/en"

	"github.com/rusq/slackdump/v2/logger"
)

// Config is the option set for the Session.
type Config struct {
	Limits Limits

	DumpFiles bool // will we save the conversation files?

	CacheDir  string // cache directory
	UserCache CacheConfig

	BaseLocation string // base location for the dump files
	Logger       logger.Interface
}

// CacheConfig represents the options for the cache.
type CacheConfig struct {
	Filename string
	MaxAge   time.Duration
	Disabled bool
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
var DefOptions = Config{
	Limits: Limits{
		Workers:         4, // number of parallel goroutines downloading files.
		DownloadRetries: 3, // this shouldn't even happen, as we have no limiter on files download.
		Tier2: TierLimits{
			Boost:   20, // seems to work fine with this boost
			Burst:   1,  // limiter will wait indefinitely if it is less than 1.
			Retries: 20, // see issue #28, sometimes slack is being difficult
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
	DumpFiles:    false,
	UserCache:    CacheConfig{Filename: "users.cache", MaxAge: 4 * time.Hour},
	CacheDir:     "",             // default cache dir
	Logger:       logger.Default, // default logger is the... default logger
	BaseLocation: ".",            // default location is the current directory
}

var (
	cfgValidator *validator.Validate // options validator
	// OptErrTranslations is the english translations for the validation
	// errors.
	OptErrTranslations ut.Translator
)

func init() {
	// initialise the config validator
	cfgValidator = validator.New()
	english := en.New()
	uni := ut.New(english, english)
	var ok bool
	OptErrTranslations, ok = uni.GetTranslator("en")
	if !ok {
		panic("internal error: failed to init translator")
	}
	if err := translations.RegisterDefaultTranslations(cfgValidator, OptErrTranslations); err != nil {
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
	return cfgValidator.Struct(o)
}

func apply[T comparable](this *T, other T) {
	if !isZero(other) && !(*this == other) {
		*this = other
	}
}

func isZero(a any) bool {
	return a == reflect.Zero(reflect.TypeOf(a)).Interface()
}
