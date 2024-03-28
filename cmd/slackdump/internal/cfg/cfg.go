// Package cfg contains common configuration variables.
package cfg

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/rusq/osenv/v2"

	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/auth/browser"
	"github.com/rusq/slackdump/v3/logger"
)

const (
	filenameLayout = "20060102_150405"
)

var (
	TraceFile string
	LogFile   string
	Verbose   bool

	Output     string
	ConfigFile string
	Workspace  string

	SlackToken   string
	SlackCookie  string
	LoginTimeout time.Duration = browser.DefLoginTimeout
	Limits                     = slackdump.DefLimits
	// playwright stuff
	Browser         browser.Browser
	LegacyBrowser   bool
	ForceEnterprise bool

	DownloadFiles bool
	NoChunkCache  bool

	// Oldest is the default timestamp of the oldest message to fetch, that is
	// used by the dump and export commands.
	Oldest = TimeValue(time.Time{})
	// Latest is the default timestamp of the newest message to fetch, that is
	// used by the dump and export commands.  It is set to an exact value
	// for the dump to be consistent.
	Latest = TimeValue(time.Now())

	LocalCacheDir      string
	UserCacheRetention time.Duration
	NoUserCache        bool

	Log logger.Interface
)

type FlagMask uint16

const (
	DefaultFlags  FlagMask = 0
	OmitAuthFlags FlagMask = 1 << iota
	OmitDownloadFlag
	OmitConfigFlag
	OmitOutputFlag
	OmitCacheDir
	OmitWorkspaceFlag
	OmitUserCacheFlag
	OmitTimeframeFlag
	OmitChunkCacheFlag

	OmitAll = OmitConfigFlag |
		OmitDownloadFlag |
		OmitOutputFlag |
		OmitCacheDir |
		OmitWorkspaceFlag |
		OmitAuthFlags |
		OmitUserCacheFlag |
		OmitTimeframeFlag |
		OmitChunkCacheFlag
)

// SetBaseFlags sets base flags
// TODO: tests.
func SetBaseFlags(fs *flag.FlagSet, mask FlagMask) {
	fs.StringVar(&TraceFile, "trace", os.Getenv("TRACE_FILE"), "trace `filename`")
	fs.StringVar(&LogFile, "log", os.Getenv("LOG_FILE"), "log `file`, if not specified, messages are printed to STDERR")
	fs.BoolVar(&Verbose, "v", osenv.Value("DEBUG", false), "verbose messages")

	if mask&OmitAuthFlags == 0 {
		fs.StringVar(&SlackToken, "token", osenv.Secret("SLACK_TOKEN", ""), "Slack `token`")
		// COOKIE environment variable is deprecated and will be removed in v2.5.0, use SLACK_COOKIE instead.
		fs.StringVar(&SlackCookie, "cookie", osenv.Secret("SLACK_COOKIE", osenv.Secret("COOKIE", "")), "d= cookie `value` or a path to a cookie.txt file\n(environment: SLACK_COOKIE)")
		fs.Var(&Browser, "browser", "browser to use for EZ-Login 3000 (default: firefox)")
		fs.DurationVar(&LoginTimeout, "browser-timeout", LoginTimeout, "Browser login `timeout`")
		fs.BoolVar(&LegacyBrowser, "legacy-browser", false, "use legacy browser automation (playwright) for EZ-Login 3000")
		fs.BoolVar(&ForceEnterprise, "enterprise", false, "enable Enteprise workarounds")
	}
	if mask&OmitDownloadFlag == 0 {
		fs.BoolVar(&DownloadFiles, "files", true, "enables file attachments (to disable, specify: -files=false)")
	}
	if mask&OmitConfigFlag == 0 {
		fs.StringVar(&ConfigFile, "api-config", "", "configuration `file` with Slack API limits overrides.\nYou can generate one with default values with 'slackdump config new`")
	}
	if mask&OmitOutputFlag == 0 {
		base := fmt.Sprintf("slackdump_%s.zip", time.Now().Format(filenameLayout))
		fs.StringVar(&Output, "o", osenv.Value("BASE_LOC", base), "a `location` (a directory or a ZIP file) on the local disk to save\ndownloaded files to.")
	}
	if mask&OmitCacheDir == 0 {
		fs.StringVar(&LocalCacheDir, "cache-dir", osenv.Value("CACHE_DIR", CacheDir()), "cache `directory` location\n")
	} else {
		// If the OmitCacheDir is specified, then the CacheDir will end up being
		// the default value, which is "". Therefore, we need to init the
		// cache directory.
		LocalCacheDir = CacheDir()
	}
	if mask&OmitWorkspaceFlag == 0 {
		fs.StringVar(&Workspace, "workspace", osenv.Value("SLACK_WORKSPACE", ""), "Slack workspace to use") // TODO: load from configuration.
	}
	if mask&OmitUserCacheFlag == 0 {
		fs.BoolVar(&NoUserCache, "no-user-cache", false, "disable user cache (file cache)")
		fs.DurationVar(&UserCacheRetention, "user-cache-retention", 60*time.Minute, "user cache retention duration.  After this time, the cache is considered stale and will be refreshed.")
	}
	if mask&OmitChunkCacheFlag == 0 {
		// ChunkCache can decrease the time of conversion for the archives
		// with large channels.  Caching is pretty useless for small archives.
		fs.BoolVar(&NoChunkCache, "no-chunk-cache", false, "disable chunk cache (uses temporary directory)")
	}
	if mask&OmitTimeframeFlag == 0 {
		fs.Var(&Oldest, "time-from", "timestamp of the oldest message to fetch (UTC timezone)")
		fs.Var(&Latest, "time-to", "timestamp of the newest message to fetch (UTC timezone)")
		fs.Var(&Oldest, "date-from", "timestamp of the oldest message to fetch (UTC timezone) (DEPRECATED)")
		fs.Var(&Latest, "date-to", "timestamp of the newest message to fetch (UTC timezone) (DEPRECATED)")
	}
}
