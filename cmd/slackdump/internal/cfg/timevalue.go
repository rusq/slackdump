package cfg

import (
	"flag"
	"time"

	"github.com/rusq/slackdump/v3/internal/structures"
)

// TimeValue satisfies flag.Value, used for command line parsing.
type TimeValue time.Time

var _ flag.Value = &TimeValue{}

func (tv TimeValue) String() string {
	t := time.Time(tv)
	if t.IsZero() {
		return ""
	}
	if t.Truncate(24 * time.Hour).Equal(t) {
		return t.Format(structures.DateLayout)
	}
	return t.Format(structures.TimeLayout)
}

func (tv *TimeValue) Set(s string) error {
	if s == "" {
		*tv = TimeValue(time.Time{})
		return nil
	}
	if t, err := structures.TimeParse(s); err != nil {
		return err
	} else {
		*tv = TimeValue(t)
	}
	return nil
}
