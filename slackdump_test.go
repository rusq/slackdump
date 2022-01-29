package slackdump

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/rusq/slackdump/internal/mock_os"
	"github.com/slack-go/slack"
	"golang.org/x/time/rate"
)

const (
	testRateLimit       = 100.0                 // per second
	maxRunDurationError = 10 * time.Millisecond // maximum deviation of run duration
)

func Test_maxStringLength(t *testing.T) {
	type args struct {
		strings []string
	}
	tests := []struct {
		name       string
		args       args
		wantMaxlen int
	}{
		{"ascii", args{[]string{"123", "abc", "defg"}}, 4},
		{"unicode", args{[]string{"сообщение1", "проверка", "тест"}}, 10},
		{"empty", args{[]string{}}, 0},
		{"several empty", args{[]string{"", "", "", ""}}, 0},
		{"several empty one full", args{[]string{"", "", "1", ""}}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotMaxlen := maxStringLength(tt.args.strings); gotMaxlen != tt.wantMaxlen {
				t.Errorf("maxStringLength() = %v, want %v", gotMaxlen, tt.wantMaxlen)
			}
		})
	}
}

func Test_fromSlackTime(t *testing.T) {
	type args struct {
		timestamp string
	}
	tests := []struct {
		name    string
		args    args
		want    time.Time
		wantErr bool
	}{
		{"good time", args{"1534552745.065949"}, time.Date(2018, 8, 18, 0, 39, 05, 65949, time.UTC), false},
		{"time without millis", args{"0"}, time.Date(1970, 1, 1, 0, 00, 00, 0, time.UTC), false},
		{"invalid time", args{"x"}, time.Time{}, true},
		{"invalid time", args{"x.x"}, time.Time{}, true},
		{"invalid time", args{"4.x"}, time.Time{}, true},
		{"invalid time", args{"x.4"}, time.Time{}, true},
		{"invalid time", args{".4"}, time.Time{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fromSlackTime(tt.args.timestamp)
			if (err != nil) != tt.wantErr {
				t.Errorf("fromSlackTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fromSlackTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

// calcRunDuration is the convenience function to calculate the expected run duration.
func calcRunDuration(rateLimit float64, attempts int) time.Duration {
	return time.Duration(attempts) * time.Duration(float64(time.Second)/rateLimit)
}

func Test_withRetry(t *testing.T) {
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
				rate.NewLimiter(0.0, 1), // no rate limit on this one, only delay by slack.RateLimitedError.RetryAfter
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
			if err := withRetry(tt.args.ctx, tt.args.l, tt.args.maxAttempts, tt.args.fn); (err != nil) != tt.wantErr {
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

func dAbs(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
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

func Test_validateFileStats(t *testing.T) {
	type args struct {
		maxAge time.Duration
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(mfi *mock_os.MockFileInfo)
		wantErr  bool
	}{
		{
			"ok",
			args{5 * time.Hour},
			func(mfi *mock_os.MockFileInfo) {
				mfi.EXPECT().IsDir().Return(false)
				mfi.EXPECT().Size().Return(int64(42))
				mfi.EXPECT().ModTime().Return(time.Now())
			},
			false,
		},
		{
			"is dir",
			args{5 * time.Hour},
			func(mfi *mock_os.MockFileInfo) {
				mfi.EXPECT().IsDir().Return(true)
			},
			true,
		},
		{
			"too smol",
			args{5 * time.Hour},
			func(mfi *mock_os.MockFileInfo) {
				mfi.EXPECT().IsDir().Return(false)
				mfi.EXPECT().Size().Return(int64(0))
			},
			true,
		},
		{
			"too old",
			args{5 * time.Hour},
			func(mfi *mock_os.MockFileInfo) {
				mfi.EXPECT().IsDir().Return(false)
				mfi.EXPECT().Size().Return(int64(42))
				mfi.EXPECT().ModTime().Return(time.Now().Add(-10 * time.Hour))
			},
			true,
		},
		{
			"disabled",
			args{0 * time.Hour},
			func(mfi *mock_os.MockFileInfo) {
				mfi.EXPECT().IsDir().Return(false)
				mfi.EXPECT().Size().Return(int64(42))
				mfi.EXPECT().ModTime().Return(time.Now().Add(-1 * time.Nanosecond))
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mfi := mock_os.NewMockFileInfo(ctrl)

			tt.expectFn(mfi)

			if err := validateFileStats(mfi, tt.args.maxAge); (err != nil) != tt.wantErr {
				t.Errorf("validateFileStats() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
