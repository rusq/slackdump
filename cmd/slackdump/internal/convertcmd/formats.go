package convertcmd

import (
	"fmt"
	"strings"
)

// datafmt is an enumeration of supported data formats.
//
//go:generate stringer -type=datafmt -trimprefix=L
type datafmt uint8

const (
	Fdump datafmt = iota
	Fexport
	Fchunk
)

func (e *datafmt) Set(v string) error {
	v = strings.ToLower(v)
	for i := 0; i < len(_datafmt_index)-1; i++ {
		if strings.ToLower(_datafmt_name[_datafmt_index[i]:_datafmt_index[i+1]]) == v {
			*e = datafmt(i)
			return nil
		}
	}
	return fmt.Errorf("unknown format: %s", v)
}
