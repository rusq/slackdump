package slackdump

import (
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/time/rate"
)

// tier represents rate limit tier:
// https://api.slack.com/docs/rate-limits
type slackTier int8

const (
	// defined as events per minute
	tier1 slackTier = 1
	tier2 slackTier = 20
	tier3 slackTier = 50
	tier4 slackTier = 100
)

// newLimiter returns throttler with rateLimit requests per minute
func newLimiter(st slackTier) *rate.Limiter {
	callsPerSec := float64(st) / 60.0
	l := rate.NewLimiter(rate.Limit(callsPerSec), 1)
	return l
}

func fromSlackTime(timestamp string) (time.Time, error) {
	strTime := strings.Split(timestamp, ".")
	var hi, lo int64

	hi, err := strconv.ParseInt(strTime[0], 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	if len(strTime) > 1 {
		lo, err = strconv.ParseInt(strTime[1], 10, 64)
		if err != nil {
			return time.Time{}, err
		}
	}
	t := time.Unix(hi, lo).UTC()
	return t, nil
}

func maxStringLength(strings []string) (maxlen int) {
	for i := range strings {
		l := utf8.RuneCountInString(strings[i])
		if l > maxlen {
			maxlen = l
		}
	}
	return
}
