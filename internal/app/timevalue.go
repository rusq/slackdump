package app

import "time"

type TimeValue time.Time

const timeFmt = "2006-01-02T15:04:05"

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
	if t, err := time.Parse(s, timeFmt); err != nil {
		return err
	} else {
		*tv = TimeValue(t)
	}
	return nil
}
