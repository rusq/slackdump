package network

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime/trace"
	"sync"
	"time"

	"github.com/rusq/slack"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v3/logger"
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

// ErrRetryFailed is returned if number of retry attempts exceeded the retry attempts limit and
// function wasn't able to complete without errors.
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
func WithRetry(ctx context.Context, lim *rate.Limiter, maxAttempts int, fn func() error) error {
	var ok bool
	if maxAttempts == 0 {
		maxAttempts = defNumAttempts
	}

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

		cbErr := fn()
		if cbErr == nil {
			ok = true
			break
		}
		lastErr = cbErr

		tracelogf(ctx, "error", "WithRetry: %[1]s (%[1]T) after %[2]d attempts", cbErr, attempt+1)
		var (
			rle *slack.RateLimitedError
			sce slack.StatusCodeError
			ne  *net.OpError // read tcp error: see #234
		)
		switch {
		case errors.As(cbErr, &rle):
			tracelogf(ctx, "info", "got rate limited, sleeping %s (%s)", rle.RetryAfter, cbErr)
			time.Sleep(rle.RetryAfter)
			continue
		case errors.As(cbErr, &sce):
			if isRecoverable(sce.Code) {
				// possibly transient error
				delay := waitFn(attempt)
				tracelogf(ctx, "info", "got server error %d, sleeping %s (%s)", sce.Code, delay, cbErr)
				time.Sleep(delay)
				continue
			}
		case errors.As(cbErr, &ne):
			if ne.Op == "read" || ne.Op == "write" {
				// possibly transient error
				delay := netWaitFn(attempt)
				tracelogf(ctx, "info", "got network error %s on %q, sleeping %s", cbErr, ne.Op, delay)
				time.Sleep(delay)
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

func tracelogf(ctx context.Context, category string, fmt string, a ...any) {
	mu.RLock()
	defer mu.RUnlock()
	lg := logger.FromContext(ctx)
	trace.Logf(ctx, category, fmt, a...)
	lg.Debugf(fmt, a...)
}

// SetMaxAllowedWaitTime sets the maximum time to wait for a transient error.
func SetMaxAllowedWaitTime(d time.Duration) {
	mu.Lock()
	defer mu.Unlock()

	maxAllowedWaitTime = d
}
