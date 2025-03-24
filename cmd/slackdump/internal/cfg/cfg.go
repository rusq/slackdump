// Package cfg contains common configuration variables.
package cfg

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/rusq/osenv/v2"

	"github.com/rusq/slackdump/v3/internal/network"
)

const (
	filenameLayout = "20060102_150405"
)

var (
	TraceFile   string
	LogFile     string
	JsonHandler bool
	Verbose     bool

	Output     string
	ConfigFile string
	Workspace  string

	Limits = network.DefLimits

	ForceEnterprise bool
	MachineIDOvr    string // Machine ID override
	NoEncryption    bool   // disable encryption

	MemberOnly       bool
	OnlyChannelUsers bool

	WithFiles   bool
	WithAvatars bool
	RecordFiles bool // record file chunks in chunk files.

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
	NoChunkCache       bool
	UseChunkFiles      bool // Use chunk files for storage, instead of sqlite database.

	Log *slog.Logger = slog.Default()
	// LoadSecrets is a flag that indicates whether to load secrets from the
	// environment variables.
	LoadSecrets bool

	Version BuildInfo // version propagated by main package.
)

type BuildInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

func (b BuildInfo) String() string {
	return fmt.Sprintf("Slackdump %s (commit: %s) built on: %s", b.Version, b.Commit, b.Date)
}

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
	OmitCustomUserFlags
	OmitRecordFilesFlag
	OmitDownloadAvatarsFlag
	OmitChunkFileMode

	OmitAll = OmitConfigFlag |
		OmitDownloadFlag |
		OmitOutputFlag |
		OmitCacheDir |
		OmitWorkspaceFlag |
		OmitAuthFlags |
		OmitUserCacheFlag |
		OmitTimeframeFlag |
		OmitChunkCacheFlag |
		OmitCustomUserFlags |
		OmitRecordFilesFlag |
		OmitDownloadAvatarsFlag |
		OmitChunkFileMode
)

// SetBaseFlags sets base flags
func SetBaseFlags(fs *flag.FlagSet, mask FlagMask) {
	setDevFlags(fs, mask) // no op if not in dev mode.
	fs.StringVar(&TraceFile, "trace", os.Getenv("TRACE_FILE"), "trace `filename`")
	fs.StringVar(&LogFile, "log", os.Getenv("LOG_FILE"), "log `file`, if not specified, messages are printed to STDERR")
	fs.BoolVar(&JsonHandler, "log-json", osenv.Value("JSON_LOG", false), "log in JSON format")
	fs.BoolVar(&Verbose, "v", osenv.Value("DEBUG", false), "verbose messages")

	if mask&OmitAuthFlags == 0 {
		fs.BoolVar(&ForceEnterprise, "enterprise", false, "enable Enteprise module, you need to specify this option if you're using Slack Enterprise Grid")
		fs.BoolVar(&LoadSecrets, "load-env", false, "load secrets from the environment, .env, .env.txt or secrets.txt file")
	}
	if mask&OmitAuthFlags == 0 || mask&OmitCacheDir == 0 {
		// machine-id flag will be automatically enabled if auth flags or cache dir flags are enabled.
		fs.StringVar(&MachineIDOvr, "machine-id", osenv.Secret("MACHINE_ID_OVERRIDE", ""), "override the machine ID for encryption")
		fs.BoolVar(&NoEncryption, "no-encryption", osenv.Value("DISABLE_ENCRYPTION", false), "disable encryption for cache and credential files")
	}
	if mask&OmitDownloadFlag == 0 {
		fs.BoolVar(&WithFiles, "files", true, "enables file attachments download (to disable, specify: -files=false)")
		if mask&OmitRecordFilesFlag == 0 {
			fs.BoolVar(&RecordFiles, "files-rec", false, "include file chunks in chunk files")
		}
	}
	if mask&OmitDownloadAvatarsFlag == 0 {
		fs.BoolVar(&WithAvatars, "avatars", false, "enables user avatar download (placed in __avatars directory)")
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
		fs.StringVar(&Workspace, "workspace", osenv.Value("SLACK_WORKSPACE", ""), "Slack workspace override (if not specified, the \"current\" workspace is used)")
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
	}
	if mask&OmitCustomUserFlags == 0 {
		fs.BoolVar(&MemberOnly, "member-only", false, "export only channels, which the current user belongs to (if no channels are specified)")
		fs.BoolVar(&OnlyChannelUsers, "channel-users", false, "export only users involved in the channel, and skip fetching of all users")
	}
	if mask&OmitChunkFileMode == 0 {
		fs.BoolVar(&UseChunkFiles, "legacy", false, "use chunk files for data storage instead of sqlite database (incompatible with resuming)")
	}
}
