//go:build windows
// +build windows

package network

import "time"

const maxRunDurationError = 100 * time.Millisecond // so special
