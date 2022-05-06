package main

import (
	"os"

	"github.com/rusq/slackdump/v2/auth"
)

type slackCreds struct {
	token  string
	cookie string
}

// authProvider returns the appropriate auth Provider depending on the values
// of the token and cookie.
func (c slackCreds) authProvider() (auth.Provider, error) {
	if c.token == "" {
		return auth.NewBrowser()
	}
	if c.cookie == "" {
		return nil, auth.ErrNoCookies
	}
	if isExistingFile(c.cookie) {
		return auth.NewFileCreds(c.token, c.cookie)
	}
	return auth.NewValueCreds(c.token, c.cookie)
}

func isExistingFile(name string) bool {
	fi, err := os.Stat(name)
	return err == nil && !fi.IsDir()
}
