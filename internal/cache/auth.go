// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package cache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"runtime/trace"
	"strings"

	"github.com/rusq/encio"

	"github.com/rusq/slackdump/v4/auth"
)

const ezLogin = "EZ-Login 3000"

//go:generate mockgen -source=auth.go -destination=../mocks/mock_cache/mock_cache.go Credentials,createOpener
//go:generate mockgen -destination=../mocks/mock_io/mock_io.go io ReadCloser,WriteCloser

// isWSL is true if we're running in the WSL environment
var isWSL = os.Getenv("WSL_DISTRO_NAME") != ""

// AuthData is the authentication data.
type AuthData struct {
	Token         string
	Cookie        string
	UsePlaywright bool
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
	ATPlaywright
)

// Type returns the authentication type that should be used for the current
// slack creds.  If the auth type wasn't tested on the system that the slackdump
// is being executed on it will return the valid type and ErrNotTested, so that
// this unfortunate fact could be relayed to the end-user.  If the type of the
// authentication determined is not supported for the current system, it will
// return ErrUnsupported.
func (c AuthData) Type(context.Context) (AuthType, error) {
	ez := ATRod
	if c.UsePlaywright {
		ez = ATPlaywright
	}
	if !c.IsEmpty() {
		if exists(c.Cookie) {
			return ATCookieFile, nil
		}
		return ATValue, nil
	}

	if !ezLoginSupported() {
		return ATInvalid, ErrUnsupported
	}
	if !ezLoginTested() {
		return ez, ErrNotTested
	}
	return ez, nil
}

func (c AuthData) IsEmpty() bool {
	return c.Token == "" || (auth.IsClientToken(c.Token) && c.Cookie == "")
}

// AuthProvider returns the appropriate auth Provider depending on the values
// of the token and cookie.
func (c AuthData) AuthProvider(ctx context.Context, workspace string, opts ...auth.Option) (auth.Provider, error) {
	authType, err := c.Type(ctx)
	if err != nil {
		return nil, err
	}
	workspace = strings.ToLower(workspace)

	opts = append([]auth.Option{auth.BrowserWithWorkspace(workspace)}, opts...)

	switch authType {
	case ATCookieFile:
		return auth.NewCookieFileAuth(c.Token, c.Cookie)
	case ATValue:
		return auth.NewValueAuth(c.Token, c.Cookie)
	case ATRod:
		return auth.NewRODAuth(ctx, opts...)
	case ATPlaywright:
		return auth.NewPlaywrightAuth(ctx, opts...)
	}
	return nil, errors.New("internal error: unsupported auth type")
}

func exists(name string) bool {
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

type Credentials interface {
	IsEmpty() bool
	AuthProvider(ctx context.Context, workspace string, opts ...auth.Option) (auth.Provider, error)
}

type authenticator struct {
	ct  createOpener
	dir string
}

// authOption is the authenticator option.
type authOption func(*authenticator)

func withNoEncryption() authOption {
	return func(a *authenticator) {
		a.ct = plainFile{}
	}
}

func withMachineIDOverride(id string) authOption {
	return func(a *authenticator) {
		a.ct = encryptedFile{machineID: id}
	}
}

func newAuthenticator(cacheDir string, opt ...authOption) authenticator {
	a := authenticator{
		dir: cacheDir,
		ct:  encryptedFile{},
	}
	for _, o := range opt {
		o(&a)
	}
	return a
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
// operating system on which the credentials storage was created. Optionally it
// can be overridden by providing a machine ID override to [newAuthenticator].
func (a authenticator) initProvider(ctx context.Context, filename string, workspace string, creds Credentials, opts ...auth.Option) (auth.Provider, error) {
	ctx, task := trace.NewTask(ctx, "initProvider")
	defer task.End()

	credsFile := filename
	if a.dir != "" {
		if err := os.MkdirAll(a.dir, 0o700); err != nil {
			return nil, fmt.Errorf("failed to create cache directory:  %w", err)
		}
		credsFile = filepath.Join(a.dir, filename)
	}

	// try to load the existing credentials, if saved earlier.
	lg := slog.With("cache_dir", a.dir, "filename", filename, "workspace", workspace)
	if creds == nil || creds.IsEmpty() {
		if prov, err := a.tryLoad(ctx, credsFile); err != nil {
			msg := fmt.Sprintf("failed to load saved credentials: %s", err)
			trace.Log(ctx, "warn", msg)
			slog.DebugContext(ctx, msg)
			if auth.IsInvalidAuthErr(err) {
				lg.InfoContext(ctx, "authentication details expired, relogin is necessary")
			}
		} else {
			msg := "loaded saved credentials"
			lg.Debug(msg)
			trace.Log(ctx, "info", msg)
			return prov, nil
		}
	}

	// init the authentication provider
	trace.Log(ctx, "info", "getting credentials from file or browser")
	provider, err := creds.AuthProvider(ctx, strings.ToLower(workspace), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialise the auth provider: %w", err)
	}

	if err := saveCreds(a.ct, credsFile, provider); err != nil {
		trace.Logf(ctx, "error", "failed to save credentials to: %s", credsFile)
	}

	return provider, nil
}

var authTester = (auth.Provider).Test

func (a authenticator) tryLoad(ctx context.Context, filename string) (auth.Provider, error) {
	prov, err := loadCreds(a.ct, filename)
	if err != nil {
		return nil, err
	}
	// test the loaded credentials
	if _, err := authTester(prov, ctx); err != nil {
		return nil, err
	}
	return prov, nil
}

var ErrFailed = errors.New("failed to load stored credentials")

// loadCreds loads the encrypted credentials from the file.
func loadCreds(ct createOpener, filename string) (auth.Provider, error) {
	f, err := ct.Open(filename)
	if err != nil {
		return nil, ErrFailed
	}
	defer f.Close()

	p, err := auth.Load(f)
	if err != nil {
		slog.Debug("failed to load credentials, possibly mismatched machine ID", "err", err)
		return nil, ErrFailed
	}
	return p, nil
}

// saveCreds encrypts and saves the credentials.
func saveCreds(ct createOpener, filename string, p auth.Provider) error {
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

// createOpener is the interface to operate with credentials createOpener.
type createOpener interface {
	Create(filename string) (io.WriteCloser, error)
	Open(filename string) (io.ReadCloser, error)
}

var _ createOpener = encryptedFile{}

// encryptedFile is the encrypted file wrapper.
type encryptedFile struct {
	// machineID is the machine ID override. If it is empty, the actual machine
	// ID is used.
	machineID string
}

func (f encryptedFile) Open(filename string) (io.ReadCloser, error) {
	var opts []encio.Option
	if f.machineID != "" {
		opts = append(opts, encio.WithID(f.machineID))
	}
	return encio.Open(filename, opts...)
}

func (f encryptedFile) Create(filename string) (io.WriteCloser, error) {
	var opts []encio.Option
	if f.machineID != "" {
		opts = append(opts, encio.WithID(f.machineID))
	}
	return encio.Create(filename, opts...)
}

type plainFile struct{}

func (plainFile) Create(filename string) (io.WriteCloser, error) {
	return os.Create(filename)
}

func (plainFile) Open(filename string) (io.ReadCloser, error) {
	return os.Open(filename)
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
