//go:build !debug

package cfg

import "flag"

const (
	CPUProfile = ""
	MEMProfile = ""
)

func setDevFlags(fs *flag.FlagSet, mask FlagMask) {}
