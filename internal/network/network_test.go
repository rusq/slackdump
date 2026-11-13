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
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"testing/synctest"
	"time"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v4/internal/fixtures"
)

const (
	testRateLimit = 100.0 // per second
)

// calcRunDuration is the convenience function to calculate the expected run
// duration.
func calcRunDuration(rateLimit float64, attempts int) time.Duration {
	return time.Duration(attempts) * time.Duration(float64(time.Second)/rateLimit)
}

func calcExpRunDuration(attempts int) time.Duration {
	var sec time.Duration
	for i := range attempts {
		sec += expWait(i)
	}
	return sec
}

// errRateFnFn will return slack.RateLimitedError for numAttempts time and err
// after.
func errRateFnFn(numAttempts int, retryAfter time.Duration, err error) func(ctx context.Context) error {
	i := 0
	return func(ctx context.Context) error {
		if i < numAttempts {
			i++
			return &slack.RateLimitedError{RetryAfter: retryAfter}
		}
		return err
	}
}

// errSeqFn will return err for forTimes time and thenErr after.
func errSeqFn(err error, forTimes int, thenErr error) func(ctx context.Context) error {
	i := 0
	return func(ctx context.Context) error {
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

func TestValidateSlackFileURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{name: "Slack file URL", url: "https://files.slack.com/files-pri/T1-F1/download/file.txt", want: true},
		{name: "GovSlack file URL", url: "https://files.slack-gov.com/files-pri/T1-F1/download/file.txt", want: true},
		{name: "other valid Slack file path", url: "https://files.slack.com/archives/T1/p-123/file.txt", want: true},
		{name: "mixed case host and scheme", url: "HTTPS://FILES.SLACK.COM/file.txt", want: true},
		{name: "explicit HTTPS port", url: "https://files.slack.com:443/file.txt", want: true},
		{name: "base Slack domain", url: "https://slack.com/file.txt"},
		{name: "workspace subdomain", url: "https://workspace.slack.com/file.txt"},
		{name: "API subdomain", url: "https://api.slack.com/file.txt"},
		{name: "HTTP", url: "http://files.slack.com/file.txt"},
		{name: "non-default port", url: "https://files.slack.com:8443/file.txt"},
		{name: "empty port", url: "https://files.slack.com:/file.txt"},
		{name: "suffix trick", url: "https://files.slack.com.attacker.example/file.txt"},
		{name: "userinfo trick", url: "https://files.slack.com@attacker.example/file.txt"},
		{name: "IP address", url: "https://127.0.0.1/file.txt"},
		{name: "missing host", url: "https:///file.txt"},
		{name: "local path", url: "files/F1/file.txt"},
		{name: "empty", url: ""},
		{name: "malformed URL", url: "://files.slack.com/file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSlackFileURL(tt.url)
			if (err == nil) != tt.want {
				t.Fatalf("ValidateSlackFileURL(%q) error = %v, want valid = %v", tt.url, err, tt.want)
			}
			if !tt.want && !errors.Is(err, ErrUntrustedFileURL) {
				t.Fatalf("ValidateSlackFileURL(%q) error = %v, want ErrUntrustedFileURL", tt.url, err)
			}
		})
	}
}

func TestValidateSlackRedirectURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{name: "Slack file host", url: "https://files.slack.com/files-pri/T1-F1/file.txt", want: true},
		{name: "workspace host", url: "https://team.slack.com/files-pri/T1-F1/file.txt", want: true},
		{name: "enterprise host", url: "https://org.enterprise.slack.com/files-pri/T1-F1/file.txt", want: true},
		{name: "GovSlack workspace host", url: "https://team.slack-gov.com/files-pri/T1-F1/file.txt", want: true},
		{name: "base Slack domain", url: "https://slack.com/files-pri/T1-F1/file.txt", want: true},
		{name: "explicit HTTPS port", url: "https://team.slack.com:443/file.txt", want: true},
		{name: "suffix trick", url: "https://team.slack.com.attacker.example/file.txt"},
		{name: "lookalike domain", url: "https://evilslack.com/file.txt"},
		{name: "other domain", url: "https://attacker.example/file.txt"},
		{name: "HTTP", url: "http://team.slack.com/file.txt"},
		{name: "non-default port", url: "https://team.slack.com:8443/file.txt"},
		{name: "userinfo trick", url: "https://team.slack.com@attacker.example/file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSlackRedirectURL(tt.url)
			if (err == nil) != tt.want {
				t.Fatalf("ValidateSlackRedirectURL(%q) error = %v, want valid = %v", tt.url, err, tt.want)
			}
			if !tt.want && !errors.Is(err, ErrUntrustedFileURL) {
				t.Fatalf("ValidateSlackRedirectURL(%q) error = %v, want ErrUntrustedFileURL", tt.url, err)
			}
		})
	}
}

func TestIsLocalFileReference(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		want bool
	}{
		{name: "empty", ref: "", want: false},
		{name: "relative dump path", ref: "C1/attachments/F1-file.txt", want: true},
		{name: "percent in name", ref: "C1/F1-100%.txt", want: true},
		{name: "Windows path", ref: `C:\x\y.txt`, want: true},
		{name: "HTTPS URL", ref: "https://files.slack.com/file.txt", want: false},
		{name: "file URL", ref: "file:///etc/passwd", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsLocalFileReference(tt.ref); got != tt.want {
				t.Errorf("IsLocalFileReference(%q) = %v, want %v", tt.ref, got, tt.want)
			}
		})
	}
}

func TestWithRetry(t *testing.T) {
	t.Parallel()
	type args struct {
		ctx         context.Context
		l           *rate.Limiter
		maxAttempts int
		fn          func(context.Context) error
	}
	tests := []struct {
		name           string
		args           args
		wantErr        bool
		mustCompleteIn time.Duration // approximate runtime duration (within 2% threshold)
	}{
		{
			"no errors",
			args{
				t.Context(),
				rate.NewLimiter(testRateLimit, 1),
				3,
				func(ctx context.Context) error {
					return nil
				},
			},
			false,
			calcRunDuration(testRateLimit, 1), // 1/100 sec
		},
		{
			"generic error",
			args{
				t.Context(),
				rate.NewLimiter(testRateLimit, 1),
				3,
				func(ctx context.Context) error {
					return errors.New("it was at this moment he knew:  he fucked up")
				},
			},
			true,
			calcRunDuration(testRateLimit, 1),
		},
		{
			"3 retries, no error",
			args{
				t.Context(),
				rate.NewLimiter(testRateLimit, 1),
				3,
				errRateFnFn(2, 1*time.Millisecond, nil),
			},
			false,
			calcRunDuration(testRateLimit, 2),
		},
		{
			"3 retries, error on the second attempt",
			args{
				t.Context(),
				rate.NewLimiter(testRateLimit, 1),
				3,
				errRateFnFn(2, 1*time.Millisecond, errors.New("boo boo")),
			},
			true,
			calcRunDuration(testRateLimit, 2),
		},
		{
			"rate limiter test 4 limited attempts, 100 ms each",
			args{
				t.Context(),
				rate.NewLimiter(10.0, 1),
				5,
				errRateFnFn(4, 1*time.Millisecond, nil),
			},
			false,
			calcRunDuration(10.0, 4),
		},
		{
			"should honour the value in the rate limit error",
			args{
				t.Context(),
				rate.NewLimiter(1000, 1),
				5,
				errRateFnFn(4, 100*time.Millisecond, nil),
			},
			false,
			calcRunDuration(10.0, 4),
		},
		{
			"running out of retries",
			args{
				t.Context(),
				rate.NewLimiter(10.0, 1),
				5,
				errRateFnFn(100, 1*time.Millisecond, nil),
			},
			true,
			calcRunDuration(10.0, 4),
		},
		{
			"network error (#234)",
			args{
				t.Context(),
				rate.NewLimiter(10.0, 1),
				3,
				errSeqFn(&net.OpError{Op: "read", Err: errors.New("network error")}, 2, nil),
			},
			false,
			calcExpRunDuration(2),
		},
		{
			"callback requests a retry",
			args{
				t.Context(),
				rate.NewLimiter(10.0, 1),
				3,
				errSeqFn(ErrRetryPlease, 2, nil),
			},
			false,
			calcRunDuration(10.0, 2),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			start := time.Now()
			if err := WithRetry(tt.args.ctx, tt.args.l, tt.args.maxAttempts, tt.args.fn); (err != nil) != tt.wantErr {
				t.Errorf("withRetry() error = %v, wantErr %v", err, tt.wantErr)
			}
			runTime := time.Since(start)
			ξ := dAbs(runTime - tt.mustCompleteIn)
			t.Logf("runtime = %s, mustCompleteIn = %s, ξ = |%[1]s - %[2]s| = %[3]s", runTime, tt.mustCompleteIn, ξ)
			if ξ > maxRunDurationError {
				t.Errorf("runtime error %s is not within allowed threshold: %s", ξ, maxRunDurationError)
			}
		})
	}
	// setting fast wait function
	t.Run("500 error handling", func(t *testing.T) {
		fixtures.SkipOnWindows(t)
		setWaitFunc(func(attempt int) time.Duration { return 50 * time.Millisecond })
		t.Cleanup(func() { setWaitFunc(cubicWait) })

		codes := []int{500, 502, 503, 504, 598}
		for _, code := range codes {
			thisCode := code
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
				err := WithRetry(t.Context(), rate.NewLimiter(1, 1), testRetryCount, func(ctx context.Context) error {
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
				if dur < wait(testRetryCount-1)-waitThreshold || wait(testRetryCount-1)+waitThreshold < dur {
					t.Errorf("expected duration to be around %s, got %s", wait(testRetryCount), dur)
				}
			})
		}
		t.Run("404 error", func(t *testing.T) {
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
			err := WithRetry(t.Context(), rate.NewLimiter(1, 1), testRetryCount, func(ctx context.Context) error {
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
	})
	t.Run("EOF error", func(t *testing.T) {
		setWaitFunc(func(attempt int) time.Duration { return 50 * time.Millisecond })
		t.Cleanup(func() { setWaitFunc(cubicWait) })

		reterr := []error{io.EOF, io.EOF, nil}
		var retries int

		ctx := t.Context()
		err := WithRetry(ctx, rate.NewLimiter(1, 1), 3, func(ctx context.Context) error {
			err := reterr[retries]
			if err != nil {
				retries++
			}
			return err
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, retries)
	})
	t.Run("Unexpected EOF error", func(t *testing.T) {
		setWaitFunc(func(attempt int) time.Duration { return 50 * time.Millisecond })
		t.Cleanup(func() { setWaitFunc(cubicWait) })

		reterr := []error{io.ErrUnexpectedEOF, io.ErrUnexpectedEOF, nil}
		var retries int

		ctx := t.Context()
		err := WithRetry(ctx, rate.NewLimiter(1, 1), 3, func(ctx context.Context) error {
			err := reterr[retries]
			if err != nil {
				retries++
			}
			return err
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, retries)
	})
	t.Run("meaningful error message", func(t *testing.T) {
		setWaitFunc(func(attempt int) time.Duration { return 50 * time.Millisecond })
		t.Cleanup(func() { setWaitFunc(cubicWait) })
		errFunc := func(ctx context.Context) error {
			return slack.StatusCodeError{Code: 500, Status: "Internal Server Error"}
		}
		err := WithRetry(t.Context(), rate.NewLimiter(1, 1), 1, errFunc)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		assert.ErrorContains(t, err, "Internal Server Error")
		assert.ErrorIs(t, err, &ErrRetryFailed{})
		var sce slack.StatusCodeError
		assert.ErrorAs(t, errors.Unwrap(err), &sce)
	})
	t.Run("read-only Slack errors", func(t *testing.T) {
		setWaitFunc(func(int) time.Duration { return time.Millisecond })
		t.Cleanup(func() { setWaitFunc(cubicWait) })

		t.Run("HTTP 200 Slack failures are retried", func(t *testing.T) {
			for _, code := range []string{"fatal_error", "internal_error", "service_unavailable", "invalid_auth"} {
				t.Run(code, func(t *testing.T) {
					calls := 0
					ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls++
						assert.Equal(t, "token", r.FormValue("token"))
						if calls == 1 {
							fmt.Fprintf(w, `{"ok":false,"error":%q}`, code)
							return
						}
						fmt.Fprint(w, `{"ok":true,"messages":[]}`)
					}))
					defer ts.Close()
					client := slack.New("token", slack.OptionAPIURL(ts.URL+"/"))
					err := WithRetry(t.Context(), rate.NewLimiter(rate.Inf, 1), 3, func(ctx context.Context) error {
						_, err := client.GetConversationHistoryContext(ctx, &slack.GetConversationHistoryParameters{})
						return err
					}, RetryReadOnly)
					assert.NoError(t, err)
					assert.Equal(t, 2, calls)
				})
			}
		})
		t.Run("wrapped value failure retries after delay", func(t *testing.T) {
			setWaitFunc(func(int) time.Duration { return 20 * time.Millisecond })
			defer setWaitFunc(func(int) time.Duration { return time.Millisecond })
			calls := 0
			start := time.Now()
			err := WithRetry(t.Context(), rate.NewLimiter(rate.Inf, 1), 2, func(context.Context) error {
				calls++
				if calls == 1 {
					return fmt.Errorf("wrapped: %w", slack.SlackErrorResponse{Err: "fatal_error"})
				}
				return nil
			}, RetryReadOnly)
			assert.NoError(t, err)
			assert.Equal(t, 2, calls)
			assert.GreaterOrEqual(t, time.Since(start), 20*time.Millisecond)
		})
		t.Run("exhaustion preserves wrapped error", func(t *testing.T) {
			original := fmt.Errorf("request: %w", slack.SlackErrorResponse{Err: "fatal_error"})
			calls := 0
			err := WithRetry(t.Context(), rate.NewLimiter(rate.Inf, 1), 3, func(context.Context) error { calls++; return original }, RetryReadOnly)
			assert.ErrorIs(t, err, original)
			assert.ErrorIs(t, err, &ErrRetryFailed{})
			assert.Equal(t, 3, calls)
		})
		t.Run("invalid auth retry bounds and delay", func(t *testing.T) {
			setWaitFunc(func(attempt int) time.Duration { return time.Duration(attempt+1) * time.Millisecond })
			defer setWaitFunc(func(int) time.Duration { return time.Millisecond })
			for _, tc := range []struct{ maxAttempts, wantCalls int }{{10, 10}, {0, 3}, {3, 3}, {2, 2}, {1, 1}} {
				t.Run(fmt.Sprint(tc.maxAttempts), func(t *testing.T) {
					synctest.Test(t, func(t *testing.T) {
						original := fmt.Errorf("request: %w", slack.SlackErrorResponse{Err: "invalid_auth"})
						start := time.Now()
						var calls []time.Duration
						err := WithRetry(context.Background(), rate.NewLimiter(rate.Inf, 1), tc.maxAttempts, func(context.Context) error {
							calls = append(calls, time.Since(start))
							return original
						}, RetryReadOnly)
						assert.ErrorIs(t, err, original)
						assert.ErrorIs(t, err, &ErrRetryFailed{})
						assert.Len(t, calls, tc.wantCalls)
						for i, elapsed := range calls {
							assert.Equal(t, time.Duration(i*(i+1)/2)*time.Millisecond, elapsed)
						}
						assert.Equal(t, time.Duration(tc.wantCalls*(tc.wantCalls+1)/2)*time.Millisecond, time.Since(start))
					})
				})
			}
		})
		t.Run("invalid auth respects configured limit with mixed errors", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				calls := 0
				original := fmt.Errorf("request: %w", slack.SlackErrorResponse{Err: "fatal_error"})
				err := WithRetry(context.Background(), rate.NewLimiter(rate.Inf, 1), 10, func(context.Context) error {
					calls++
					if calls == 1 {
						return slack.SlackErrorResponse{Err: "invalid_auth"}
					}
					return original
				}, RetryReadOnly)
				assert.Equal(t, 10, calls)
				assert.ErrorIs(t, err, original)
				assert.ErrorIs(t, err, &ErrRetryFailed{})
			})
		})
		t.Run("auth errors fail fast under applicable policies", func(t *testing.T) {
			for _, code := range []string{"invalid_auth", "token_expired", "token_revoked", "not_authed"} {
				for _, readOnly := range []bool{false, true} {
					if code == "invalid_auth" && readOnly {
						continue
					}
					t.Run(fmt.Sprintf("%s/readOnly=%t", code, readOnly), func(t *testing.T) {
						calls := 0
						original := fmt.Errorf("request: %w", slack.SlackErrorResponse{Err: code})
						var policies []RetryPolicy
						if readOnly {
							policies = append(policies, RetryReadOnly)
						}
						err := WithRetry(t.Context(), rate.NewLimiter(rate.Inf, 1), 3, func(context.Context) error { calls++; return original }, policies...)
						assert.ErrorIs(t, err, original)
						assert.Equal(t, 1, calls)
					})
				}
			}
		})
		t.Run("general retry does not retry Slack API errors", func(t *testing.T) {
			calls := 0
			original := fmt.Errorf("request: %w", slack.SlackErrorResponse{Err: "fatal_error"})
			err := WithRetry(t.Context(), rate.NewLimiter(rate.Inf, 1), 3, func(context.Context) error { calls++; return original })
			assert.ErrorIs(t, err, original)
			assert.Equal(t, 1, calls)
		})
		t.Run("cancellation interrupts delay", func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			calls := 0
			err := WithRetry(ctx, rate.NewLimiter(rate.Inf, 1), 3, func(context.Context) error {
				calls++
				cancel()
				return slack.SlackErrorResponse{Err: "fatal_error"}
			}, RetryReadOnly)
			assert.ErrorIs(t, err, context.Canceled)
			assert.Equal(t, 1, calls)
		})
		t.Run("invalid auth cancellation interrupts delay", func(t *testing.T) {
			setWaitFunc(func(int) time.Duration { return 5 * time.Second })
			defer setWaitFunc(func(int) time.Duration { return time.Millisecond })
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				go func() { time.Sleep(2 * time.Second); cancel() }()
				start := time.Now()
				calls := 0
				err := WithRetry(ctx, rate.NewLimiter(rate.Inf, 1), 3, func(context.Context) error {
					calls++
					return slack.SlackErrorResponse{Err: "invalid_auth"}
				}, RetryReadOnly)
				assert.ErrorIs(t, err, context.Canceled)
				assert.Equal(t, 1, calls)
				assert.Equal(t, 2*time.Second, time.Since(start))
			})
		})
	})
}

func Test_cubicWait(t *testing.T) {
	t.Parallel()
	type args struct {
		attempt int
	}
	tests := []struct {
		name string
		args args
		want time.Duration
	}{
		{"attempt 0", args{0}, 1 * time.Second},
		{"attempt 1", args{1}, 8 * time.Second},
		{"attempt 2", args{2}, 27 * time.Second},
		{"attempt 4", args{4}, 125 * time.Second},
		{"attempt 5", args{5}, 216 * time.Second},
		{"attempt 6", args{6}, maxAllowedWaitTime}, // check if capped properly
		{"attempt 100", args{1000}, maxAllowedWaitTime},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := cubicWait(tt.args.attempt); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("waitTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isRecoverable(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			if got := isRecoverable(tt.args.statusCode); got != tt.want {
				t.Errorf("isRecoverable() = %v, want %v", got, tt.want)
			}
		})
	}
}
