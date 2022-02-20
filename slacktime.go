package slackdump

// in this file: slack timestamp parsing functions

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// parseThreadID parses the thread id (ie. p1577694990000400) and returns
// time.Time.
func parseThreadID(threadID string) (time.Time, error) {
	if len(threadID) == 0 || threadID[0] != 'p' {
		return time.Time{}, errors.New("not a thread ID")
	}
	if _, err := strconv.ParseInt(threadID[1:], 10, 64); err != nil {
		return time.Time{}, errors.New("invalid thread ID")
	}
	return parseSlackTS(threadID[1:11] + "." + threadID[11:])
}

func parseSlackTS(timestamp string) (time.Time, error) {
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

func formatSlackTS(ts time.Time) string {
	hi := ts.Unix()
	lo := ts.UnixNano() % 100000
	return fmt.Sprintf("%d.%06d", hi, lo)
}
