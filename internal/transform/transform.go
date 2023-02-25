// Package transform provides a set of functions to transform a given Slack
// event record into an output file or archive format file.
package transform

import "fmt"

type EventsInfo struct {
	ChannelID    string
	File         string
	FilesDir     string
	IsCompressed bool
}

func (ei EventsInfo) String() string {
	return fmt.Sprintf("channel %q, file %q, filesdir %q, compressed %v", ei.ChannelID, ei.File, ei.FilesDir, ei.IsCompressed)
}
