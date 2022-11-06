// Package cfg contains common configuration variables.
package cfg

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/rusq/osenv/v2"

	"github.com/rusq/slackdump/v2"
)

var (
	TraceFile string
	LogFile   string
	Verbose   bool

	ConfigFile string
	BaseLoc    string // base location - directory or a zip file.
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
	OmitCacheDir
	OmitWorkspaceFlag

	OmitAll = OmitConfigFlag |
		OmitDownloadFlag |
		OmitBaseLoc |
		OmitCacheDir |
		OmitWorkspaceFlag |
		OmitAuthFlags
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
		fs.StringVar(&ConfigFile, "api-config", "", "configuration `file` with Slack API limits overrides.\nYou can generate one with default values with 'slackdump config new`")
	}
	if mask&OmitBaseLoc == 0 {
		base := fmt.Sprintf("slackdump_%s.zip", time.Now().Format("20060102_150304"))
		fs.StringVar(&BaseLoc, "base", osenv.Value("BASE_LOC", base), "a `location` (directory or a ZIP file) on a local disk where the files will be saved.")
	}
	if mask&OmitCacheDir == 0 {
		fs.StringVar(&SlackOptions.CacheDir, "cache-dir", osenv.Value("CACHE_DIR", CacheDir()), "cache `directory` location")
	}
	if mask&OmitWorkspaceFlag == 0 {
		fs.StringVar(&Workspace, "workspace", osenv.Value("SLACK_WORKSPACE", ""), "Slack workspace to use") // TODO: load from configuration.
	}
}
