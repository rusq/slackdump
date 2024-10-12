//go:build !386

package fasttime

import "strconv"

// As int on 64-bit systems is 64 bit, it is possible to use faster Atoi.

var atoi = strconv.Atoi
