package main

import (
	"errors"
	"os"
	"runtime"

	"github.com/rusq/dlog"
	"github.com/rusq/slackdump/v2/auth"
)

type slackCreds struct {
	token  string
	cookie string
}

// authProvider returns the appropriate auth Provider depending on the values
// of the token and cookie.
func (c slackCreds) authProvider() (auth.Provider, error) {
	if c.token == "" || c.cookie == "" {
		if !ezLoginSupported() {
			return nil, errors.New("EZ-Login 3000 is not supported on this OS, please use the manual login method")
		}
		if !ezLoginTested() {
			dlog.Println("warning, EZ-Login 3000 is not tested on this OS, if it doesn't work, use manual login method")
		}
		return auth.NewBrowserAuth()
	}
	if isExistingFile(c.cookie) {
		return auth.NewCookieFileAuth(c.token, c.cookie)
	}
	return auth.NewValueAuth(c.token, c.cookie)
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
