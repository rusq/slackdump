// Package cfg contains common configuration variables.
package cfg

import (
	"flag"
	"os"
)

var (
	TraceFile string
)

// SetBaseFlags sets base flags
func SetBaseFlags(fs *flag.FlagSet) {
	fs.StringVar(&TraceFile, "trace", os.Getenv("TRACE_FILE"), "trace `filename`")
}
