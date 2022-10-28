package workspace

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
)

var flagmask = cfg.OmitAll

var CmdWorkspace = &base.Command{
	Run:       nil,
	Wizard:    nil,
	UsageLine: "slackdump workspace",
	Short:     "authenticate or choose workspace to run on",
	Long: `
Slackdump supports working with multiple Slack Workspaces without the need
to authenticate again (unless login credentials are expired).

Workspace command allows to authenticate in a new Slack Workspace, list already
authenticated workspaces, and choose a workspace that you have previously
authenticated.

Run:

	slackdump help login

To learn more about different login options.

Workspaces are stored in cache directory on this device:
` + cfg.CacheDir() + `
`,
	CustomFlags: false,
	FlagMask:    flagmask,
	PrintFlags:  false,
	RequireAuth: false,
	Commands:    []*base.Command{CmdListWsp},
}

var once sync.Once

const (
	defaultWspFilename = "provider.bin"
	currentWspFile     = "workspace.txt"
)

// Current returns the current workspace name, if present.  It only returns
// an error, if it fails to create a cache directory.  If the current workspace
// file is not found, it returns empty string and nil error.
// The cache directory is created with rwx------ permissions, if it does not
// exist.
func Current() (string, error) {
	var err error
	once.Do(func() {
		err = os.MkdirAll(cfg.CacheDir(), 0700)
	})
	if err != nil {
		return "", err
	}
	f, err := os.Open(filepath.Join(cfg.CacheDir(), currentWspFile))
	if err != nil {
		return defaultWspFilename, nil
	}
	defer f.Close()

	return readWsp(f), nil
}

func readWsp(r io.Reader) string {
	var current string
	if _, err := fmt.Fscanln(r, &current); err != nil {
		return defaultWspFilename
	}
	return strings.TrimSpace(current)
}
