package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/rusq/dlog"
	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/internal/encio"
)

const (
	credsFile = "provider.bin"
)

type SlackCreds struct {
	Token  string
	Cookie string
}

var (
	ErrNotTested   = errors.New("warning, EZ-Login 3000 is not tested on this OS, if it doesn't work, use manual login method")
	ErrUnsupported = errors.New("EZ-Login 3000 is not supported on this OS, please use the manual login method")
)

// Type returns the authentication type that should be used for the current
// slack creds.  If the auth type wasn't tested on the system that the slackdump
// is being executed on it will return the valid type and ErrNotTested, so that
// this unfortunate fact could be relayed to the end-user.  If the type of the
// authentication determined is not supported for the current system, it will
// return ErrUnsupported.
func (c SlackCreds) Type(ctx context.Context) (auth.Type, error) {
	if !c.IsEmpty() {
		if isExistingFile(c.Cookie) {
			return auth.TypeCookieFile, nil
		}
		return auth.TypeValue, nil
	}

	if !ezLoginSupported() {
		return auth.TypeInvalid, ErrUnsupported
	}
	if !ezLoginTested() {
		return auth.TypeBrowser, ErrNotTested
	}
	return auth.TypeBrowser, nil

}

func (c SlackCreds) IsEmpty() bool {
	return c.Token == "" || c.Cookie == ""
}

// AuthProvider returns the appropriate auth Provider depending on the values
// of the token and cookie.
func (c SlackCreds) AuthProvider(ctx context.Context, workspace string) (auth.Provider, error) {
	authType, err := c.Type(ctx)
	if err != nil {
		return nil, err
	}
	switch authType {
	case auth.TypeBrowser:
		return auth.NewBrowserAuth(ctx, auth.BrowserWithWorkspace(workspace))
	case auth.TypeCookieFile:
		return auth.NewCookieFileAuth(c.Token, c.Cookie)
	case auth.TypeValue:
		return auth.NewValueAuth(c.Token, c.Cookie)
	}
	return nil, errors.New("internal error: unsupported auth type")
}

func isExistingFile(name string) bool {
	fi, err := os.Stat(name)
	return err == nil && !fi.IsDir()
}

func ezLoginSupported() bool {
	return runtime.GOARCH != "386"
}

func ezLoginTested() bool {
	switch runtime.GOOS {
	default:
		return false
	case "windows", "linux", "darwin":
		return true
	}
}

func InitProvider(ctx context.Context, cacheDir string, workspace string, creds SlackCreds) (auth.Provider, error) {
	credsLoc := filepath.Join(cacheDir, credsFile)

	lg := dlog.FromContext(ctx)
	// try to load the existing credentials, if saved earlier.
	if creds.IsEmpty() {
		prov, err := loadCreds(credsLoc)
		if err != nil {
			lg.Debugf("failed to load credentials: %s", err)
		} else {
			if err := slackdump.TestAuth(ctx, prov); err == nil {
				// authenticated with the saved creds.
				lg.Print("loaded saved credentials")
				return prov, nil
			}
			lg.Debugf("no stored credentials on the system")
			// fallthrough to getting the credentials from auth provider
		}
	}

	// init the authentication provider
	provider, err := creds.AuthProvider(ctx, workspace)
	if err != nil {
		return nil, fmt.Errorf("failed to initialise the auth provider: %w", err)
	}

	if err := saveCreds(credsLoc, provider); err != nil {
		lg.Printf("warning:  failed to save credentials to: %s", credsLoc)
	}

	return provider, nil
}

var errLoadFailed = errors.New("failed to load stored credentials")

func loadCreds(filename string) (auth.Provider, error) {
	f, err := encio.Open(filename)
	if err != nil {
		return nil, errLoadFailed
	}
	defer f.Close()

	return auth.Load(f)
}

func saveCreds(filename string, p auth.Provider) error {
	f, err := encio.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return auth.Save(f, p)
}

func AuthReset(cacheDir string) error {
	return os.Remove(filepath.Join(cacheDir, credsFile))
}
