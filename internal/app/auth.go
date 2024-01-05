package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/trace"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/auth/browser"
	"github.com/rusq/slackdump/v2/internal/encio"
)

//go:generate mockgen -source=auth.go -destination=../mocks/mock_app/mock_app.go Credentials,createOpener
//go:generate mockgen -destination=../mocks/mock_io/mock_io.go io ReadCloser,WriteCloser

const (
	credsFile = "provider.bin"
)

// isWSL is true if we're running in the WSL environment
var isWSL = os.Getenv("WSL_DISTRO_NAME") != ""

// SlackCreds holds the Token and Cookie reference.
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
func (c SlackCreds) Type(ctx context.Context, legacy bool) (auth.Type, error) {
	var browserAuth = auth.TypeRod
	if legacy {
		browserAuth = auth.TypeBrowser
	}
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
		return browserAuth, ErrNotTested
	}
	return browserAuth, nil

}

func (c SlackCreds) IsEmpty() bool {
	return c.Token == "" || (auth.IsClientToken(c.Token) && c.Cookie == "")
}

// AuthProvider returns the appropriate auth Provider depending on the values
// of the token and cookie.
func (c SlackCreds) AuthProvider(ctx context.Context, workspace string, browser browser.Browser, legacy bool) (auth.Provider, error) {
	authType, err := c.Type(ctx, legacy)
	if err != nil {
		return nil, err
	}
	switch authType {
	case auth.TypeRod:
		return auth.NewRODAuth(ctx, auth.BrowserWithWorkspace(workspace))
	case auth.TypeBrowser:
		return auth.NewBrowserAuth(ctx, auth.BrowserWithWorkspace(workspace), auth.BrowserWithBrowser(browser))
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
	return runtime.GOARCH != "386" && !isWSL
}

func ezLoginTested() bool {
	switch runtime.GOOS {
	default:
		return false
	case "windows", "linux", "darwin":
		return true
	}
}

// filer is openCreator that will be used by InitProvider.
var filer createOpener = encryptedFile{}

type Credentials interface {
	IsEmpty() bool
	AuthProvider(ctx context.Context, workspace string, browser browser.Browser, legacy bool) (auth.Provider, error)
}

// InitProvider initialises the auth.Provider depending on provided slack
// credentials.  It returns auth.Provider or an error.  The logic diagram is
// available in the doc/diagrams/auth_flow.puml.
//
// If the creds is empty, it attempts to load the stored credentials.  If it
// finds them, it returns an initialised credentials provider.  If not - it
// returns the auth provider according to the type of credentials determined
// by creds.AuthProvider, and saves them to an AES-256-CFB encrypted storage.
//
// The storage is encrypted using the hash of the unique machine-ID, supplied
// by the operating system (see package encio), it makes it impossible to
// transfer and use the stored credentials on another machine (including
// virtual), even another operating system on the same machine, unless it's a
// clone of the source operating system on which the credentials storage was
// created.
func InitProvider(ctx context.Context, cacheDir string, workspace string, creds Credentials, browser browser.Browser, legacy bool) (auth.Provider, error) {
	ctx, task := trace.NewTask(ctx, "InitProvider")
	defer task.End()

	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create cache directory:  %w", err)
	}

	credsLoc := filepath.Join(cacheDir, credsFile)

	// try to load the existing credentials, if saved earlier.
	if creds.IsEmpty() {
		if prov, err := tryLoad(ctx, credsLoc); err != nil {
			trace.Logf(ctx, "warn", "no saved credentials: %s", err)
		} else {
			trace.Log(ctx, "info", "loaded saved credentials")
			return prov, nil
		}
	}

	// init the authentication provider
	trace.Log(ctx, "info", "getting credentals from file or browser")
	provider, err := creds.AuthProvider(ctx, workspace, browser, legacy)
	if err != nil {
		return nil, fmt.Errorf("failed to initialise the auth provider: %w", err)
	}

	if err := saveCreds(filer, credsLoc, provider); err != nil {
		trace.Logf(ctx, "error", "failed to save credentials to: %s", credsLoc)
	}

	return provider, nil
}

var authTester = slackdump.TestAuth

func tryLoad(ctx context.Context, filename string) (auth.Provider, error) {
	prov, err := loadCreds(filer, filename)
	if err != nil {
		return nil, err
	}
	// test the loaded credentials
	if err := authTester(ctx, prov); err != nil {
		return nil, err
	}
	return prov, nil
}

// loadCreds loads the encrypted credentials from the file.
func loadCreds(opener createOpener, filename string) (auth.Provider, error) {
	f, err := opener.Open(filename)
	if err != nil {
		return nil, errors.New("failed to load stored credentials")
	}
	defer f.Close()

	return auth.Load(f)
}

// saveCreds encrypts and saves the credentials.
func saveCreds(opener createOpener, filename string, p auth.Provider) error {
	f, err := opener.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return auth.Save(f, p)
}

// AuthReset removes the cached credentials.
func AuthReset(cacheDir string) error {
	return os.Remove(filepath.Join(cacheDir, credsFile))
}

// createOpener is the interface to be able to switch between encrypted file
// and plain text file, if needed.
type createOpener interface {
	Create(string) (io.WriteCloser, error)
	Open(string) (io.ReadCloser, error)
}

type encryptedFile struct{}

func (encryptedFile) Open(filename string) (io.ReadCloser, error) {
	return encio.Open(filename)
}

func (encryptedFile) Create(filename string) (io.WriteCloser, error) {
	return encio.Create(filename)
}
