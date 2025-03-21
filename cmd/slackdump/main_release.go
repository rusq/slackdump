//go:build !debug

package main

// initDebug does some debug mode initialisations.  It is a noop in non-debug builds.
func initDebug() {}
