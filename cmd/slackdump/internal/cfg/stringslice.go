package cfg

import (
	"flag"
	"fmt"
	"slices"
	"strings"

	"github.com/rusq/slackdump/v3"
)

const stringSliceSep = ","

// StringSlice provides a flag.Value interface for a slice of strings.
type StringSlice []string

var _ flag.Value = new(StringSlice)

func (ss *StringSlice) Set(s string) error {
	parts := strings.Split(s, stringSliceSep)
	*ss = parts
	return nil
}

func (ss *StringSlice) String() string {
	return strings.Join(*ss, stringSliceSep)
}

type slackChanTypes StringSlice

func (ss *slackChanTypes) Set(s string) error {
	(*StringSlice)(ss).Set(s)
	for _, v := range *ss {
		if !slices.Contains(slackdump.AllChanTypes, v) {
			return fmt.Errorf("allowed values are: %v", slackdump.AllChanTypes)
		}
	}
	return nil
}

func (ss *slackChanTypes) String() string {
	return (*StringSlice)(ss).String()
}
