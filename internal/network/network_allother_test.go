//go:build !windows
// +build !windows

package network

import "time"

const maxRunDurationError = 10 * time.Millisecond // maximum deviation of run duration
