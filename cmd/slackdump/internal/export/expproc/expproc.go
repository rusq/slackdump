// Package expproc implements the export processor interface.  The processor
// is responsible for writing the data to disk.  It does many things
// concurrently.
//
// GOOD LUCK DEBUGGING THIS.
package expproc

import "github.com/slack-go/slack"

// channelName returns the channel name, or the channel ID if it is a DM.
func channelName(ch *slack.Channel) string {
	if ch.IsIM {
		return ch.ID
	}
	return ch.Name
}
