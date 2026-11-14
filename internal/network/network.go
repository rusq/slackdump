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
//
// No backoff is applied after the final attempt, except for the rate limit:
// its Retry-After is always honoured, so that the caller's next request is
// not sent while the API is still throttling it.  If the context is cancelled
// by the time attempts run out, the cancellation cause is returned instead of
// [ErrRetryFailed].
func WithRetry(ctx context.Context, lim *rate.Limiter, maxAttempts int, fn func(ctx context.Context) error) error {
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
		if cbErr == nil {
			// success
			return nil
		}
		// Record every failure, including a requested retry, so that an
		// earlier error is not reported as the final one.
		lastErr = cbErr
		if errors.Is(cbErr, ErrRetryPlease) {
			// callback requested a retry
			lg.DebugContext(ctx, "retry requested", "attempt", attempt+1)
			continue
		}

		if !strings.EqualFold(cbErr.Error(), "pagination complete") && !errors.Is(cbErr, context.Canceled) {
			// let the upstream report the error, if necessary.
			lg.DebugContext(ctx, "WithRetry", "error", cbErr, "attempt", attempt+1)
		}

		delay, waitOnLast, ok := backoff(cbErr, attempt)
		if !ok {
			return fmt.Errorf("callback error: %w", cbErr)
		}
		if attempt == maxAttempts-1 && !waitOnLast {
			// no further attempts, backing off would only delay the return.
			break
		}
		slog.WarnContext(ctx, "transient error, sleeping", "error", cbErr, "delay", delay.String(), "attempt", attempt+1)
		tracelogf(ctx, "info", "transient error, sleeping %s (%s)", delay, cbErr)
		if err := SleepCtx(ctx, delay); err != nil {
			return err
		}
	}
	if cause := context.Cause(ctx); cause != nil {
		return cause
	}
	return &ErrRetryFailed{Err: lastErr}
}

// backoff returns the delay before the next attempt after err on the given
// attempt, and ok set to false if err is not transient and must not be
// retried.  waitOnLast is true if the delay must be honoured even after the
// final attempt, which is the case for the rate limit.
func backoff(err error, attempt int) (delay time.Duration, waitOnLast bool, ok bool) {
	var (
		rle *slack.RateLimitedError
		sce slack.StatusCodeError
		ne  *net.OpError // read tcp error: see #234
	)
	switch {
	case errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF):
		return wait(attempt), false, true
	case errors.As(err, &rle):
		return rle.RetryAfter, true, true
	case errors.As(err, &sce) && isRecoverable(sce.Code):
		return wait(attempt), false, true
	case errors.As(err, &ne) && (ne.Op == "read" || ne.Op == "write"):
		return netWait(attempt), false, true
	}
	return 0, false, false
}

// SleepCtx waits for d, or until ctx is done, in which case it returns the
// cause of the context cancellation.
func SleepCtx(ctx context.Context, d time.Duration) error {
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
