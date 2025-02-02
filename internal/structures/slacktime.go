package structures

// in this file: slack timestamp parsing functions

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
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
	const (
		base = 10
		bit  = 64
	)
	sSec, sMicro, found := strings.Cut(timestamp, ".")
	if sSec == "" {
		return time.Time{}, errors.New("empty timestamp")
	}
	var t int64
	var err error
	if !found {
		t, err = strconv.ParseInt(sSec+"000000", base, bit)
	} else {
		t, err = strconv.ParseInt(sSec+sMicro, base, bit)
	}
	if err != nil {
		return time.Time{}, err
	}
	return time.UnixMicro(t).UTC(), nil
}

func FormatSlackTS(ts time.Time) string {
	if ts.IsZero() || ts.Before(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)) {
		return ""
	}
	hi := ts.UTC().Unix()
	lo := ts.UTC().UnixMicro() % 1_000_000
	return fmt.Sprintf("%d.%06d", hi, lo)
}

func ThreadIDtoTS(threadID string) string {
	if len(threadID) == 0 || threadID[0] != 'p' {
		return ""
	}
	if _, err := strconv.ParseInt(threadID[1:], 10, 64); err != nil {
		return ""
	}
	return threadID[1:11] + "." + threadID[11:]
}
