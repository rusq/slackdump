//go:build 386

package fasttime

import "strconv"

// int size on the 32-bit systems is 32 bit (surprise), this constraints us to slower 64-bit implementation.

var atoi = func(s string) (int64, error) { return strconv.ParseInt(s, 10, 64) }
