//go:build !linux
// +build !linux

package encio

import "github.com/mzky/machineid"

var machineIDFn = machineid.ProtectedID
