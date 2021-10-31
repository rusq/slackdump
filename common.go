package slackdump

import (
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	slackTier1 = 4   //per Minute
	slackTier2 = 35  //per Minute
	slackTier3 = 70  //per Minute
	slackTier4 = 130 //per Minute
)

// getThrottler returns throttler with rateLimit requests per minute
func getThrottler(rateLimit time.Duration) <-chan time.Time {
	rate := time.Minute / rateLimit
	return time.Tick(rate)
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
