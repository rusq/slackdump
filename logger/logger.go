package logger

import (
	"log"
	"os"

	"github.com/rusq/dlog"
)

// Interface is the interface for a logger.
type Interface interface {
	Debug(...any)
	Debugf(fmt string, a ...any)
	Print(...any)
	Printf(fmt string, a ...any)
	Println(...any)
}

// Default is the default logger.  It logs to stderr and debug logging can be
// enabled by setting the DEBUG environment variable to 1.  For example:
//
//	DEBUG=1 slackdump
var Default = dlog.New(log.Default().Writer(), "", log.LstdFlags, os.Getenv("DEBUG") == "1")

// Silent is a logger that does not log anything.
var Silent = silent{}

// Silent is a logger that does not log anything.
type silent struct{}

func (s silent) Debug(...any)                {}
func (s silent) Debugf(fmt string, a ...any) {}
func (s silent) Print(...any)                {}
func (s silent) Printf(fmt string, a ...any) {}
func (s silent) Println(...any)              {}
