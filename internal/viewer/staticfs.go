package viewer

import (
	"embed"
	"io/fs"
)

//go:embed static/*
var embeddedStaticFS embed.FS

// StaticFS returns the embedded viewer static assets.
func StaticFS() fs.FS {
	sub, err := fs.Sub(embeddedStaticFS, "static")
	if err != nil {
		panic(err)
	}
	return sub
}
