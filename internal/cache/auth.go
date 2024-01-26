package cache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/trace"
	"strings"

	"github.com/rusq/encio"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/logger"
)

const ezLogin = "EZ-Login 3000"

//go:generate mockgen -source=auth.go -destination=../../mocks/mock_appauth/mock_appauth.go Credentials,createOpener
//go:generate mockgen -destination=../mocks/mock_io/mock_io.go io ReadCloser,WriteCloser

// isWSL is true if we're running in the WSL environment
var isWSL = os.Getenv("WSL_DISTRO_NAME") != ""

// SlackCreds holds the Token and Cookie reference.
type SlackCreds struct {
	Token  string
	Cookie string
}

var (
	ErrNotTested   = errors.New("warning, " + ezLogin + " is not tested on this OS, if it doesn't work, use manual login method")
	ErrUnsupported = errors.New("" + ezLogin + " is not supported on this OS, please use the manual login method")
)

type AuthType int

const (
	ATInvalid AuthType = iota
	ATValue
	ATCookieFile
	ATRod
	ATBrowser
)

// Type returns the authentication type that should be used for the current
// slack creds.  If the auth type wasn't tested on the system that the slackdump
// is being executed on it will return the valid type and ErrNotTested, so that
// this unfortunate fact could be relayed to the end-user.  If the type of the
// authentication determined is not supported for the current system, it will
// return ErrUnsupported.
func (c SlackCreds) Type(ctx context.Context) (AuthType, error) {
	if !c.IsEmpty() {
		if isExistingFile(c.Cookie) {
			return ATCookieFile, nil
		}
		return ATValue, nil
	}

	if !ezLoginSupported() {
		return ATInvalid, ErrUnsupported
	}
	if !ezLoginTested() {
		return ATRod, ErrNotTested
	}
	return ATRod, nil
}

func (c SlackCreds) IsEmpty() bool {
	return c.Token == "" || (auth.IsClientToken(c.Token) && c.Cookie == "")
}

// AuthProvider returns the appropriate auth Provider depending on the values
// of the token and cookie.
func (c SlackCreds) AuthProvider(ctx context.Context, workspace string, opts ...auth.Option) (auth.Provider, error) {
	authType, err := c.Type(ctx)
	if err != nil {
		return nil, err
	}
	workspace = strings.ToLower(workspace)

	opts = append([]auth.Option{auth.BrowserWithWorkspace(workspace)}, opts...)

	switch authType {
	case ATBrowser:
		return auth.NewBrowserAuth(ctx, opts...)
	case ATCookieFile:
		return auth.NewCookieFileAuth(c.Token, c.Cookie)
	case ATValue:
		return auth.NewValueAuth(c.Token, c.Cookie)
	case ATRod:
		return auth.NewRODAuth(ctx, opts...)
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
var filer container = encryptedFile{}

type Credentials interface {
	IsEmpty() bool
	AuthProvider(ctx context.Context, workspace string, opts ...auth.Option) (auth.Provider, error)
}

// initProvider initialises the auth.Provider depending on provided slack
// credentials.  It returns auth.Provider or an error.  The logic diagram is
// available in the doc/diagrams/auth_flow.puml.
//
// If the creds is empty, it attempts to load the stored credentials.  If it
// finds them, it returns an initialised credentials provider.  If not - it
// returns the auth provider according to the type of credentials determined
// by creds.AuthProvider, and saves them to an AES-256-CFB encrypted storage.
//
// The storage is encrypted using the hash of the unique machine-ID, supplied by
// the operating system (see package encio), it makes it impossible use the
// stored credentials on another machine (including virtual), even another
// operating system on the same machine, unless it's a clone of the source
// operating system on which the credentials storage was created.
func initProvider(ctx context.Context, cacheDir string, filename string, workspace string, creds Credentials, opts ...auth.Option) (auth.Provider, error) {
	ctx, task := trace.NewTask(ctx, "initProvider")
	defer task.End()

	credsFile := filename
	if cacheDir != "" {
		if err := os.MkdirAll(cacheDir, 0o700); err != nil {
			return nil, fmt.Errorf("failed to create cache directory:  %w", err)
		}
		credsFile = filepath.Join(cacheDir, filename)
	}

	// try to load the existing credentials, if saved earlier.
	lg := logger.FromContext(ctx)
	if creds == nil || creds.IsEmpty() {
		if prov, err := tryLoad(ctx, credsFile); err != nil {
			msg := fmt.Sprintf("failed to load saved credentials: %s", err)
			trace.Logf(ctx, "warn", msg)
			if auth.IsInvalidAuthErr(err) {
				lg.Println("authentication details expired, relogin is necessary")
			}
		} else {
			msg := "loaded saved credentials"
			lg.Debug(msg)
			trace.Log(ctx, "info", msg)
			return prov, nil
		}
	}

	// init the authentication provider
	trace.Log(ctx, "info", "getting credentals from file or browser")
	provider, err := creds.AuthProvider(ctx, strings.ToLower(workspace), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialise the auth provider: %w", err)
	}

	if err := saveCreds(filer, credsFile, provider); err != nil {
		trace.Logf(ctx, "error", "failed to save credentials to: %s", credsFile)
	}

	return provider, nil
}

var authTester = (auth.Provider).Test

func tryLoad(ctx context.Context, filename string) (auth.Provider, error) {
	prov, err := loadCreds(filer, filename)
	if err != nil {
		return nil, err
	}
	// test the loaded credentials
	if err := authTester(prov, ctx); err != nil {
		return nil, err
	}
	return prov, nil
}

// loadCreds loads the encrypted credentials from the file.
func loadCreds(ct container, filename string) (auth.Provider, error) {
	f, err := ct.Open(filename)
	if err != nil {
		return nil, errors.New("failed to load stored credentials")
	}
	defer f.Close()

	return auth.Load(f)
}

// saveCreds encrypts and saves the credentials.
func saveCreds(ct container, filename string, p auth.Provider) error {
	f, err := ct.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return auth.Save(f, p)
}

// AuthReset removes the cached credentials.
func AuthReset(cacheDir string) error {
	return os.Remove(filepath.Join(cacheDir, defCredsFile))
}

// container is the interface to operate with credentials container.
type container interface {
	Create(filename string) (io.WriteCloser, error)
	Open(filename string) (io.ReadCloser, error)
}

// encryptedFile is the encrypted file container.
type encryptedFile struct{}

func (encryptedFile) Open(filename string) (io.ReadCloser, error) {
	return encio.Open(filename)
}

func (encryptedFile) Create(filename string) (io.WriteCloser, error) {
	return encio.Create(filename)
}

// EZLoginFlags is a diagnostic function that returns the map of flags that
// describe the EZ-Login feature.
func EzLoginFlags() map[string]bool {
	return map[string]bool{
		"supported": ezLoginSupported(),
		"tested":    ezLoginTested(),
		"wsl":       isWSL,
	}
}
