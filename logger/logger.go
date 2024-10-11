package logger

import (
	"context"
	"log"
	"os"

	"github.com/rusq/dlog"
)

// Interface is the interface for a logger.
type Interface interface {
	Debug(...any)
	Debugf(fmt string, a ...any)
	Debugln(...any)
	Print(...any)
	Printf(fmt string, a ...any)
	Println(...any)
	IsDebug() bool
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
func (s silent) Debugln(...any)              {}
func (s silent) Print(...any)                {}
func (s silent) Printf(fmt string, a ...any) {}
func (s silent) Println(...any)              {}
func (s silent) IsDebug() bool               { return false }

type logCtx uint8

const (
	logCtxKey logCtx = iota
)

// NewContext returns a new context with the logger.
func NewContext(ctx context.Context, l Interface) context.Context {
	return context.WithValue(ctx, logCtxKey, l)
}

// FromContext returns the logger from the context.  If no logger is found,
// the Default logger is returned.
func FromContext(ctx context.Context) Interface {
	if l, ok := ctx.Value(logCtxKey).(Interface); ok {
		return l
	}
	return Default
}
