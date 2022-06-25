package logger

import (
	"log"
	"os"

	"github.com/rusq/dlog"
)

type Interface interface {
	Debug(...any)
	Debugf(fmt string, a ...any)
	Print(...any)
	Printf(fmt string, a ...any)
	Println(...any)
}

var Default = dlog.New(log.Default().Writer(), "", log.LstdFlags, os.Getenv("DEBUG") == "1")
