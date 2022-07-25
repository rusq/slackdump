package app

import (
	"flag"
	"time"
)

const timeFmt = "2006-01-02T15:04:05"

// TimeValue satisfies flag.Value, used for command line parsing.
type TimeValue time.Time

var _ flag.Value = &TimeValue{}

func (tv *TimeValue) String() string {
	if time.Time(*tv).IsZero() {
		return ""
	}
	return time.Time(*tv).Format(timeFmt)
}

func (tv *TimeValue) Set(s string) error {
	if s == "" {
		return nil
	}
	if t, err := time.Parse(timeFmt, s); err != nil {
		return err
	} else {
		*tv = TimeValue(t)
	}
	return nil
}
