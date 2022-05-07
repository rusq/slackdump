package network

import (
	"context"
	"fmt"
	"runtime/trace"
	"time"

	"github.com/pkg/errors"
	"github.com/rusq/dlog"
	"github.com/slack-go/slack"
	"golang.org/x/time/rate"
)

// defNumAttempts is the default number of retry attempts.
const defNumAttempts = 3

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

		err = fn()
		if err == nil {
			ok = true
			break
		}

		trace.Logf(ctx, "error", "slackRetry: %s", err)
		var rle *slack.RateLimitedError
		if !errors.As(err, &rle) {
			return errors.WithStack(err)
		}

		msg := fmt.Sprintf("got rate limited, sleeping %s", rle.RetryAfter)
		trace.Log(ctx, "info", msg)
		dlog.Debug(msg)

		time.Sleep(rle.RetryAfter)
	}
	if !ok {
		return ErrRetryFailed
	}
	return nil
}
