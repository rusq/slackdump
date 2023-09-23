package network

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

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

func calcExpRunDuration(attempts int) time.Duration {
	var sec time.Duration
	for i := 0; i < attempts; i++ {
		sec += expWait(i)
	}
	return sec
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

// errSeqFn will return err for forTimes time and thenErr after.
func errSeqFn(err error, forTimes int, thenErr error) func() error {
	i := 0
	return func() error {
		if i < forTimes {
			i++
			return err
		}
		return thenErr
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
		{"should honour the value in the rate limit error",
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
		{
			"network error (#234)",
			args{
				context.Background(),
				rate.NewLimiter(10.0, 1),
				3,
				errSeqFn(&net.OpError{Op: "read"}, 2, nil),
			},
			false,
			calcExpRunDuration(2),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			if err := WithRetry(tt.args.ctx, tt.args.l, tt.args.maxAttempts, tt.args.fn); (err != nil) != tt.wantErr {
				t.Errorf("withRetry() error = %v, wantErr %v", err, tt.wantErr)
			}
			runTime := time.Since(start)
			ξ := dAbs(runTime - tt.mustCompleteIn)
			t.Logf("runtime = %s, mustCompleteIn = %s, ξ = ABS(%[1]s - %[2]s) = %[3]s", runTime, tt.mustCompleteIn, ξ)
			if ξ > maxRunDurationError {
				t.Errorf("runtime error %s is not within allowed threshold: %s", ξ, maxRunDurationError)
			}
		})
	}
}

func Test500ErrorHandling(t *testing.T) {
	waitFn = func(attempt int) time.Duration { return 50 * time.Millisecond }
	defer func() {
		waitFn = cubicWait
	}()

	var codes = []int{500, 502, 503, 504, 598}
	for _, code := range codes {
		var thisCode = code
		// This test is to ensure that we handle 500 errors correctly.
		t.Run(fmt.Sprintf("%d error", code), func(t *testing.T) {

			const (
				testRetryCount = 1
				waitThreshold  = 100 * time.Millisecond
			)

			// Create a test server that returns a 500 error.
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(thisCode)
			}))
			defer ts.Close()

			// Create a new client with the test server as the endpoint.
			client := slack.New("token", slack.OptionAPIURL(ts.URL+"/"))

			start := time.Now()
			// Call the client with a retry.
			err := WithRetry(context.Background(), rate.NewLimiter(1, 1), testRetryCount, func() error {
				_, err := client.GetConversationHistory(&slack.GetConversationHistoryParameters{})
				if err == nil {
					return errors.New("expected error, got nil")
				}
				return err
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			dur := time.Since(start)
			if dur < waitFn(testRetryCount-1)-waitThreshold || waitFn(testRetryCount-1)+waitThreshold < dur {
				t.Errorf("expected duration to be around %s, got %s", waitFn(testRetryCount), dur)
			}

		})
	}
	t.Run("404 error", func(t *testing.T) {
		t.Parallel()

		const (
			testRetryCount = 1
		)

		// Create a test server that returns a 404 error.
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
		}))
		defer ts.Close()

		// Create a new client with the test server as the endpoint.
		client := slack.New("token", slack.OptionAPIURL(ts.URL+"/"))

		// Call the client with a retry.
		start := time.Now()
		err := WithRetry(context.Background(), rate.NewLimiter(1, 1), testRetryCount, func() error {
			_, err := client.GetConversationHistory(&slack.GetConversationHistoryParameters{})
			if err == nil {
				return errors.New("expected error, got nil")
			}
			return err
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		dur := time.Since(start)
		if dur > 500*time.Millisecond { // 404 error should not be retried
			t.Errorf("expected no sleep, but slept for %s", dur)
		}
	})
}

func Test_cubicWait(t *testing.T) {
	type args struct {
		attempt int
	}
	tests := []struct {
		name string
		args args
		want time.Duration
	}{
		{"attempt 0", args{0}, 8 * time.Second},
		{"attempt 1", args{1}, 27 * time.Second},
		{"attempt 2", args{2}, 64 * time.Second},
		{"attempt 2", args{4}, 216 * time.Second},
		{"attempt 100", args{5}, maxAllowedWaitTime}, // check if capped properly
		{"attempt 100", args{1000}, maxAllowedWaitTime},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cubicWait(tt.args.attempt); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("waitTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isRecoverable(t *testing.T) {
	type args struct {
		statusCode int
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"500", args{500}, true},
		{"502", args{502}, true},
		{"503", args{503}, true},
		{"504", args{504}, true},
		{"598", args{598}, true},
		{"599", args{599}, true},
		{"200", args{200}, false},
		{"400", args{400}, false},
		{"404", args{404}, false},
		{"408", args{408}, true},
		{"429", args{429}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRecoverable(tt.args.statusCode); got != tt.want {
				t.Errorf("isRecoverable() = %v, want %v", got, tt.want)
			}
		})
	}
}
