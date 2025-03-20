//go:build debug

package cfg

import (
	"flag"
	"os"
)

// Additional configuration variables for dev environment.
var (
	CPUProfile string
	MEMProfile string
)

func setDevFlags(fs *flag.FlagSet, mask FlagMask) {
	fs.StringVar(&CPUProfile, "cpuprofile", os.Getenv("CPU_PROFILE"), "write CPU profile to `file`")
	fs.StringVar(&MEMProfile, "memprofile", os.Getenv("MEM_PROFILE"), "write memory profile to `file`")
}
