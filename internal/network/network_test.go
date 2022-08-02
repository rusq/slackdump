package network

import (
	"context"
	"testing"
	"time"

	"errors"

	"github.com/slack-go/slack"
	"golang.org/x/time/rate"
)

const (
	testRateLimit = 100.0 // per second
)

// calcRunDuration is the convenience function to calculate the expected run duration.
func calcRunDuration(rateLimit float64, attempts int) time.Duration {
	return time.Duration(attempts) * time.Duration(float64(time.Second)/rateLimit)
}

// retryFn will return slack.RateLimitedError for numAttempts time and err after.
func retryFn(numAttempts int, retryAfter time.Duration, err error) func() error {
	i := 0
	return func() error {
		if i < numAttempts {
			i++
			return &slack.RateLimitedError{RetryAfter: retryAfter}
		}
		return err
	}
}

func dAbs(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

func Test_withRetry(t *testing.T) {
	t.Parallel()
	type args struct {
		ctx         context.Context
		l           *rate.Limiter
		maxAttempts int
		fn          func() error
	}
	tests := []struct {
		name           string
		args           args
		wantErr        bool
		mustCompleteIn time.Duration // approximate runtime duration (within 2% threshold)
	}{
		{"no errors",
			args{
				context.Background(),
				rate.NewLimiter(testRateLimit, 1),
				3,
				func() error {
					return nil
				},
			},
			false,
			calcRunDuration(testRateLimit, 1), // 1/100 sec
		},
		{"generic error",
			args{
				context.Background(),
				rate.NewLimiter(testRateLimit, 1),
				3,
				func() error {
					return errors.New("it was at this moment he knew:  he fucked up")
				},
			},
			true,
			calcRunDuration(testRateLimit, 1),
		},
		{"3 retries, no error",
			args{
				context.Background(),
				rate.NewLimiter(testRateLimit, 1),
				3,
				retryFn(2, 1*time.Millisecond, nil),
			},
			false,
			calcRunDuration(testRateLimit, 2),
		},
		{"3 retries, error on the second attempt",
			args{
				context.Background(),
				rate.NewLimiter(testRateLimit, 1),
				3,
				retryFn(2, 1*time.Millisecond, errors.New("boo boo")),
			},
			true,
			calcRunDuration(testRateLimit, 2),
		},
		{"rate limiter test 4 lmited attempts, 100 ms each",
			args{
				context.Background(),
				rate.NewLimiter(10.0, 1),
				5,
				retryFn(4, 1*time.Millisecond, nil),
			},
			false,
			calcRunDuration(10.0, 4),
		},
		{"slackRetry should honour the value in the rate limit error",
			args{
				context.Background(),
				rate.NewLimiter(1000, 1),
				5,
				retryFn(4, 100*time.Millisecond, nil),
			},
			false,
			calcRunDuration(10.0, 4),
		},
		{"running out of retries",
			args{
				context.Background(),
				rate.NewLimiter(10.0, 1),
				5,
				retryFn(100, 1*time.Millisecond, nil),
			},
			true,
			calcRunDuration(10.0, 4),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			if err := WithRetry(tt.args.ctx, tt.args.l, tt.args.maxAttempts, tt.args.fn); (err != nil) != tt.wantErr {
				t.Errorf("withRetry() error = %v, wantErr %v", err, tt.wantErr)
			}
			runTime := time.Since(start)
			runTimeError := dAbs(runTime - tt.mustCompleteIn)
			t.Logf("runtime = %s, mustCompleteIn = %s, error = ABS(%[1]s - %[2]s) = %[3]s", runTime, tt.mustCompleteIn, runTimeError)
			if runTimeError > maxRunDurationError {
				t.Errorf("runtime error %s is not within allowed threshold: %s", runTimeError, maxRunDurationError)
			}
		})
	}
}
