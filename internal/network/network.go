package network

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"runtime/trace"
	"strings"
	"sync"
	"time"

	"github.com/rusq/slack"
	"golang.org/x/time/rate"
)

// defNumAttempts is the default number of retry attempts.
const (
	defNumAttempts = 3
)

var (
	// maxAllowedWaitTime is the maximum time to wait for a transient error.
	// The wait time for a transient error depends on the current retry
	// attempt number and is calculated as: (attempt+2)^3 seconds, capped at
	// maxAllowedWaitTime.
	maxAllowedWaitTime = 5 * time.Minute

	// waitFn returns the amount of time to wait before retrying depending on
	// the current attempt.  This variable exists to reduce the test time.
	waitFn    = cubicWait
	netWaitFn = expWait

	mu sync.RWMutex
)

func setWaitFunc(fn func(int) time.Duration) {
	mu.Lock()
	defer mu.Unlock()
	waitFn = fn
}

func setNetWaitFunc(fn func(int) time.Duration) {
	mu.Lock()
	defer mu.Unlock()

	netWaitFn = fn
}

func wait(n int) time.Duration {
	mu.RLock()
	defer mu.RUnlock()
	return waitFn(n)
}

func netWait(n int) time.Duration {
	mu.RLock()
	defer mu.RUnlock()
	return netWaitFn(n)
}

// ErrRetryPlease should be returned by the callback function to indicate that
// the call should be retried.
var ErrRetryPlease = errors.New("retry")

// ErrRetryFailed is returned if number of retry attempts exceeded the retry
// attempts limit and function wasn't able to complete without errors.
type ErrRetryFailed struct {
	Err error
}

func (e *ErrRetryFailed) Error() string {
	return fmt.Sprintf("callback was unable to complete without errors within the allowed number of retries: %s", e.Err)
}

func (e *ErrRetryFailed) Unwrap() error {
	return e.Err
}

func (e *ErrRetryFailed) Is(target error) bool {
	_, ok := target.(*ErrRetryFailed)
	return ok
}

// WithRetry will run the callback function fn. If the function returns
// slack.RateLimitedError, it will delay, and then call it again up to
// maxAttempts times. It will return an error if it runs out of attempts.
func WithRetry(ctx context.Context, lim *rate.Limiter, maxAttempts int, fn func(ctx context.Context) error) error {
	var ok bool
	if maxAttempts == 0 {
		maxAttempts = defNumAttempts
	}
	lg := slog.With("maxAttempts", maxAttempts)

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// calling wait to ensure that we don't exceed the rate limit
		var err error
		trace.WithRegion(ctx, "WithRetry.wait", func() {
			err = lim.Wait(ctx)
		})
		if err != nil {
			return err
		}

		cbErr := fn(ctx)
		if errors.Is(cbErr, ErrRetryPlease) {
			// callback requested a retry
			lg.DebugContext(ctx, "retry requested", "attempt", attempt+1)
			continue
		}
		if cbErr == nil {
			// success
			ok = true
			break
		}
		lastErr = cbErr

		if !strings.EqualFold(cbErr.Error(), "pagination complete") && !errors.Is(cbErr, context.Canceled) {
			lg.ErrorContext(ctx, "WithRetry", "error", cbErr, "attempt", attempt+1)
		}
		var (
			rle *slack.RateLimitedError
			sce slack.StatusCodeError
			ne  *net.OpError // read tcp error: see #234
		)
		switch {
		case errors.Is(cbErr, io.EOF) || errors.Is(cbErr, io.ErrUnexpectedEOF):
			// EOF is a transient error
			delay := wait(attempt)
			slog.WarnContext(ctx, "got EOF, sleeping", "error", cbErr, "delay", delay.String())
			tracelogf(ctx, "info", "got EOF, sleeping %s (%s)", delay, cbErr)
			if err := sleepCtx(ctx, delay); err != nil {
				return err
			}
			slog.Debug("resuming after EOF")
			continue
		case errors.As(cbErr, &rle):
			slog.InfoContext(ctx, "got rate limited, sleeping", "retry_after_sec", rle.RetryAfter, "error", cbErr)
			tracelogf(ctx, "info", "got rate limited, sleeping %s (%s)", rle.RetryAfter, cbErr)
			if err := sleepCtx(ctx, rle.RetryAfter); err != nil {
				return err
			}
			slog.Info("resuming after rate limit")
			continue
		case errors.As(cbErr, &sce):
			if isRecoverable(sce.Code) {
				// possibly transient error
				delay := wait(attempt)
				slog.WarnContext(ctx, "got server error, sleeping", "status_code", sce.Code, "error", cbErr, "delay", delay.String())
				tracelogf(ctx, "info", "got server error %d, sleeping %s (%s)", sce.Code, delay, cbErr)
				if err := sleepCtx(ctx, delay); err != nil {
					return err
				}
				continue
			}
		case errors.As(cbErr, &ne):
			if ne.Op == "read" || ne.Op == "write" {
				// possibly transient error
				delay := netWait(attempt)
				slog.WarnContext(ctx, "got network error, sleeping", "op", ne.Op, "error", cbErr, "delay", delay.String())
				tracelogf(ctx, "info", "got network error %s on %q, sleeping %s", cbErr, ne.Op, delay)
				if err := sleepCtx(ctx, delay); err != nil {
					return err
				}
				continue
			}
		}

		return fmt.Errorf("callback error: %w", cbErr)
	}
	if !ok {
		return &ErrRetryFailed{Err: lastErr}
	}
	return nil
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}

// isRecoverable returns true if the status code is a recoverable error.
func isRecoverable(statusCode int) bool {
	return (statusCode >= http.StatusInternalServerError && statusCode <= 599 && statusCode != 501) || statusCode == 408
}

// cubicWait is the wait time function.  Time is calculated as (x+2)^3 seconds,
// where x is the current attempt number. The maximum wait time is capped at 5
// minutes.
func cubicWait(attempt int) time.Duration {
	x := attempt + 1 // this is to ensure that we sleep at least a second.
	delay := time.Duration(x*x*x) * time.Second
	if delay > maxAllowedWaitTime {
		return maxAllowedWaitTime
	}
	return delay
}

func expWait(attempt int) time.Duration {
	delay := time.Duration(2<<uint(attempt)) * time.Second
	if delay > maxAllowedWaitTime {
		return maxAllowedWaitTime
	}
	return delay
}

func tracelogf(ctx context.Context, category string, format string, a ...any) {
	mu.RLock()
	defer mu.RUnlock()
	trace.Logf(ctx, category, format, a...)
}

// SetMaxAllowedWaitTime sets the maximum time to wait for a transient error.
func SetMaxAllowedWaitTime(d time.Duration) {
	mu.Lock()
	defer mu.Unlock()

	maxAllowedWaitTime = d
}
