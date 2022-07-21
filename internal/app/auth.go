package app

import (
	"context"
	"errors"
	"os"
	"runtime"

	"github.com/rusq/slackdump/v2/auth"
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
	if c.Token == "" || c.Cookie == "" {
		if !ezLoginSupported() {
			return auth.TypeInvalid, ErrUnsupported
		}
		if !ezLoginTested() {
			return auth.TypeBrowser, ErrNotTested
		}
		return auth.TypeBrowser, nil
	}
	if isExistingFile(c.Cookie) {
		return auth.TypeCookieFile, nil
	}
	return auth.TypeValue, nil
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
