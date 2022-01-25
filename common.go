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
type tier int

const (
	// base throttling defined as events per minute
	tier1  tier = 1
	tier2  tier = 20
	tier3  tier = 50
	tier4  tier = 100
	noTier tier = 0
)

// newLimiter returns throttler with rateLimit requests per minute.
// optionally caller may specify the boost
func newLimiter(t tier, burst uint, boost int) *rate.Limiter {
	callsPerSec := float64(int(t)+boost) / 60.0
	l := rate.NewLimiter(rate.Limit(callsPerSec), int(burst))
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
