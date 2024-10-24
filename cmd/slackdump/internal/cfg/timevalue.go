package cfg

import (
	"flag"
	"time"
)

const TimeLayout = "2006-01-02T15:04:05"

// TimeValue satisfies flag.Value, used for command line parsing.
type TimeValue time.Time

var _ flag.Value = &TimeValue{}

func (tv TimeValue) String() string {
	if time.Time(tv).IsZero() {
		return ""
	}
	return time.Time(tv).Format(TimeLayout)
}

func (tv *TimeValue) Set(s string) error {
	if s == "" {
		*tv = TimeValue(time.Time{})
		return nil
	}
	if t, err := time.Parse(TimeLayout, s); err != nil {
		return err
	} else {
		*tv = TimeValue(t)
	}
	return nil
}
