package structures

// in this file: slack timestamp parsing functions

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"errors"
)

// ParseThreadID parses the thread id (ie. p1577694990000400) and returns
// time.Time.
func ParseThreadID(threadID string) (time.Time, error) {
	if len(threadID) == 0 || threadID[0] != 'p' {
		return time.Time{}, errors.New("not a thread ID")
	}
	if _, err := strconv.ParseInt(threadID[1:], 10, 64); err != nil {
		return time.Time{}, errors.New("invalid thread ID")
	}
	return ParseSlackTS(threadID[1:11] + "." + threadID[11:])
}

// ParseSlackTS parses the slack timestamp.
func ParseSlackTS(timestamp string) (time.Time, error) {
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

func FormatSlackTS(ts time.Time) string {
	if ts.IsZero() || ts.Before(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)) {
		return ""
	}
	hi := ts.Unix()
	lo := ts.UnixNano() % 1_000_000
	return fmt.Sprintf("%d.%06d", hi, lo)
}
