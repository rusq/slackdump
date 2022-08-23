package logger

import (
	"io"
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

// note: previously ioutil.Discard which is not deprecated in favord of io.Discard
// so this is valid only from go1.16
var Silent = dlog.New(io.Discard, "", log.LstdFlags, false)
