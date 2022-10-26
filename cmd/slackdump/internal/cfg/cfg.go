// Package cfg contains common configuration variables.
package cfg

import (
	"flag"
	"os"

	"github.com/rusq/osenv/v2"
)

var (
	TraceFile string
	LogFile   string
	Verbose   bool

	ConfigFile    string
	SlackToken    string
	SlackCookie   string
	DownloadFiles bool
)

type FlagMask int

const (
	DefaultFlags  FlagMask = 0
	OmitAuthFlags FlagMask = 1 << iota
	OmitDownloadFlag
	OmitConfigFlag
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
}
