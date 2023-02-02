package network

import (
	"context"
	"fmt"
	"net/http"
	"runtime/trace"
	"time"

	"errors"

	"github.com/slack-go/slack"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v2/logger"
)

// defNumAttempts is the default number of retry attempts.
const (
	defNumAttempts = 3
)

// MaxAllowedWaitTime is the maximum time to wait for a transient error.  The
// wait time for a transient error depends on the current retry attempt number
// and is calculated as: (attempt+2)^3 seconds, capped at MaxAllowedWaitTime.
var MaxAllowedWaitTime = 5 * time.Minute

// Logger is the package logger.
var Logger logger.Interface = logger.Default

// ErrRetryFailed is returned if number of retry attempts exceeded the retry attempts limit and
// function wasn't able to complete without errors.
var ErrRetryFailed = errors.New("callback was not able to complete without errors within the allowed number of retries")

// withRetry will run the callback function fn. If the function returns
// slack.RateLimitedError, it will delay, and then call it again up to
// maxAttempts times. It will return an error if it runs out of attempts.
func WithRetry(ctx context.Context, l *rate.Limiter, maxAttempts int, fn func() error) error {
	var ok bool
	if maxAttempts == 0 {
		maxAttempts = defNumAttempts
	}
	for attempt := 0; attempt < maxAttempts; attempt++ {
		var err error
		trace.WithRegion(ctx, "withRetry.wait", func() {
			err = l.Wait(ctx)
		})
		if err != nil {
			return err
		}

		cbErr := fn()
		if cbErr == nil {
			ok = true
			break
		}

		tracelogf(ctx, "error", "slackRetry: %s after %d attempts", cbErr, attempt+1)
		var (
			rle *slack.RateLimitedError
			sce slack.StatusCodeError
		)
		if errors.As(cbErr, &rle) {
			tracelogf(ctx, "info", "got rate limited, sleeping %s", rle.RetryAfter)
			time.Sleep(rle.RetryAfter)
			continue
		} else if errors.As(cbErr, &sce) {
			if sce.Code >= http.StatusInternalServerError && sce.Code <= 599 {
				// possibly transient error
				delay := waitTime(attempt)
				tracelogf(ctx, "info", "got server error %d, sleeping %s", sce.Code, delay)
				time.Sleep(delay)
				continue
			}
		}

		return fmt.Errorf("callback error: %w", cbErr)
	}
	if !ok {
		return ErrRetryFailed
	}
	return nil
}

// waitTime returns the amount of time to wait before retrying depending on
// the current attempt. The wait time is calculated as (x+2)^3 seconds, where
// x is the current attempt number. The maximum wait time is capped at 5
// minutes.
func waitTime(attempt int) time.Duration {
	x := attempt + 2 // this is to ensure that we sleep at least 8 seconds.
	delay := time.Duration(x*x*x) * time.Second
	if delay > MaxAllowedWaitTime {
		return MaxAllowedWaitTime
	}
	return delay
}

func tracelogf(ctx context.Context, category string, fmt string, a ...any) {
	trace.Logf(ctx, category, fmt, a...)
	l().Debugf(fmt, a...)
}

func l() logger.Interface {
	if Logger == nil {
		return logger.Default
	}
	return Logger
}
