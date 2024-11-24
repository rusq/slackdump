//go:build !windows
// +build !windows

package network

import "time"

const maxRunDurationError = 20 * time.Millisecond // maximum deviation of run duration
