// Package cfg contains common configuration variables.
package cfg

import (
	"flag"
	"os"

	"github.com/rusq/osenv/v2"

	"github.com/rusq/slackdump/v2"
)

var (
	TraceFile string
	LogFile   string
	Verbose   bool

	ConfigFile string
	BaseLoc    string // base location - directory or a zip file.
	cacheDir   string // cache directory
	Workspace  string

	SlackToken   string
	SlackCookie  string
	SlackOptions = slackdump.DefOptions

	DownloadFiles bool
)

type FlagMask int

const (
	DefaultFlags  FlagMask = 0
	OmitAuthFlags FlagMask = 1 << iota
	OmitDownloadFlag
	OmitConfigFlag
	OmitBaseLoc
)

// SetBaseFlags sets base flags
func SetBaseFlags(fs *flag.FlagSet, mask FlagMask) {
	fs.StringVar(&TraceFile, "trace", os.Getenv("TRACE_FILE"), "trace `filename`")
	fs.StringVar(&LogFile, "log", os.Getenv("LOG_FILE"), "log `file`, if not specified, messages are printed to STDERR")
	fs.BoolVar(&Verbose, "v", osenv.Value("DEBUG", false), "verbose messages")

	if mask&OmitAuthFlags == 0 {
		fs.StringVar(&SlackToken, "token", osenv.Secret("SLACK_TOKEN", ""), "Slack `token`")
		// COOKIE environment variable is deprecated and will be removed in v2.5.0, use SLACK_COOKIE instead.
		fs.StringVar(&SlackCookie, "cookie", osenv.Secret("SLACK_COOKIE", osenv.Secret("COOKIE", "")), "d= cookie `value` or a path to a cookie.txt file (environment: SLACK_COOKIE)")
	}
	if mask&OmitDownloadFlag == 0 {
		fs.BoolVar(&DownloadFiles, "download", true, "enables file attachments download")
	}
	if mask&OmitConfigFlag == 0 {
		fs.StringVar(&ConfigFile, "config", "", "configuration `file` with API limits overrides")
	}
	if mask&OmitBaseLoc == 0 {
		fs.StringVar(&BaseLoc, "base", os.Getenv("BASE_LOC"), "a `location` (directory or a ZIP file) on a local disk where the files will be saved.")
	}
	fs.StringVar(&cacheDir, "cache-dir", osenv.Value("CACHE_DIR", CacheDir()), "cache `directory` location")
	fs.StringVar(&Workspace, "workspace", osenv.Value("SLACK_WORKSPACE", ""), "Slack workspace to use") // TODO: load from configuration.
}
