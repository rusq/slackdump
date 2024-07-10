package fasttime

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Time time.Time

func (t *Time) UnmarshalJSON(data []byte) error {
	clean := strings.Trim(string(data), `"`)
	if clean == "" || clean == "null" {
		*t = Time{}
		return nil
	}
	ts, err := TS2int(clean)
	if err != nil {
		return err
	}
	*t = Time(Int2Time(ts))
	return nil
}

func (t Time) MarshalJSON() ([]byte, error) {
	ts := time.Time(t).UnixMicro()
	return []byte(`"` + Int2TS(ts) + `"`), nil
}

// SlackString returns the time as a slack timestamp (i.e. "1234567890.123456").
func (t Time) SlackString() string {
	return Int2TS(time.Time(t).UnixMicro())
}

var ErrNotATimestamp = errors.New("not a slack timestamp")

// TS2int converts a slack timestamp to an int64 by stripping the dot and
// converting the string to an int64.  It is useful for fast comparison.
func TS2int(ts string) (int64, error) {
	if ts == "" {
		return 0, nil
	}
	before, after, found := strings.Cut(ts, ".")
	if !found {
		return 0, fmt.Errorf("%w: %s", ErrNotATimestamp, ts)
	}
	return strconv.ParseInt(before+after, 10, 64)
}

// Int2TS converts an int64 to a slack timestamp by inserting a dot in the
// right place.
func Int2TS(ts int64) string {
	s := strconv.FormatInt(ts, 10)
	if len(s) < 7 {
		return ""
	}
	lo := s[len(s)-6:]
	hi := s[:len(s)-6]
	return hi + "." + lo
}

// Int2Time converts an int64 to a time.Time.
func Int2Time(ts int64) time.Time {
	return time.UnixMicro(ts)
}
