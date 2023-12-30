package auth

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rusq/slackauth"
	"github.com/rusq/slackdump/v2/auth/auth_ui"
	"github.com/schollz/progressbar/v3"
)

type RodAuth struct {
	simpleProvider
	opts rodOpts
}

func (p RodAuth) Type() Type {
	return TypeRod
}

type rodOpts struct {
	ui        browserAuthUIExt
	workspace string
}

type browserAuthUIExt interface {
	BrowserAuthUI
	RequestEmail(w io.Writer) (string, error)
	RequestPassword(w io.Writer, account string) (string, error)
	RequestLoginType(w io.Writer) (int, error)
}

func RodWithWorkspace(name string) Option {
	return func(o *options) {
		o.rodOpts.workspace = name
	}
}

const expectedLoginDuration = 16 * time.Second

func NewRODAuth(ctx context.Context, opts ...Option) (RodAuth, error) {
	r := RodAuth{
		opts: rodOpts{
			ui: &auth_ui.Huh{},
		},
	}
	for _, opt := range opts {
		opt(&options{
			rodOpts: &r.opts,
		})
	}
	if r.opts.workspace == "" {
		var err error
		r.opts.workspace, err = r.opts.ui.RequestWorkspace(os.Stdout)
		if err != nil {
			return r, err
		}
		if r.opts.workspace == "" {
			return r, fmt.Errorf("workspace cannot be empty")
		}
	}
	if wsp, err := auth_ui.Sanitize(r.opts.workspace); err != nil {
		return r, err
	} else {
		r.opts.workspace = wsp
	}
	resp, err := r.opts.ui.RequestLoginType(os.Stdout)
	if err != nil {
		return r, err
	}
	var sp simpleProvider
	if resp == auth_ui.LoginSSO {
		var err error
		sp.Token, sp.Cookie, err = slackauth.Browser(ctx, r.opts.workspace)
		if err != nil {
			return r, err
		}
	} else {
		var err error
		username, err := r.opts.ui.RequestEmail(os.Stdout)
		if err != nil {
			return r, err
		}
		if username == "" {
			return r, fmt.Errorf("email cannot be empty")
		}
		password, err := r.opts.ui.RequestPassword(os.Stdout, username)
		if err != nil {
			return r, err
		}
		if password == "" {
			return r, fmt.Errorf("password cannot be empty")
		}
		done, finished := fakeProgress("Logging in...", int(expectedLoginDuration.Seconds())*10, 100*time.Millisecond)
		sp.Token, sp.Cookie, err = slackauth.Headless(ctx, r.opts.workspace, username, password)
		close(done)
		<-finished
		if err != nil {
			return r, err
		}
		fmt.Fprintln(os.Stderr, "authenticated.")
	}

	return RodAuth{
		simpleProvider: sp,
	}, nil
}

// fakeProgress starts a fake spinner and returns a channel that must be closed
// once the operation completes. interval is interval between iterations. If not
// set, will default to 50ms.
func fakeProgress(title string, max int, interval time.Duration) (chan<- struct{}, <-chan struct{}) {
	if interval == 0 {
		interval = 50 * time.Millisecond
	}
	var (
		done     = make(chan struct{})
		finished = make(chan struct{})
	)
	go func() {
		bar := progressbar.NewOptions(
			max,
			progressbar.OptionSetDescription(title),
			progressbar.OptionSpinnerType(9),
		)
		t := time.NewTicker(interval)
		defer t.Stop()

		for {
			select {
			case <-done:
				bar.Finish()
				fmt.Println()
				close(finished)
				return
			case <-t.C:
				bar.Add(1)
			}
		}
	}()
	return done, finished
}
