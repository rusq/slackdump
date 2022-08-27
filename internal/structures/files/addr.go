// Package files contains some additional file logic.
package files

import (
	"errors"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/types"
)

// Root is the parent address for the topmost message chunk.
const Root = -1

// Addr is the address of the file in the messages slice.
//
//	idxMsg    - index of the message or message reply in the provided slice
//	idxParMsg - index of the parent message. If it is not equal to Root
//	            constant, then it's the address of the message:
//
//	                msg[idxMsg].File[idxFile]
//
//	            if it is not equal to Root, then it is assumed that it is
//	            an address of a message (thread) reply:
//
//	                msg[idxParMsg].ThreadReplies[idxMsg].File[idxFile]
//
//	idxFile   - index of the file in the message's file slice.
type Addr struct {
	idxMsg    int // index of the message in the messages slice
	idxParMsg int // index of the parent message, or Root (-1) if it's the address of the top level message
	idxFile   int // index of the file in the file slice.
}

// Update locates the file by address addr, and calls fn with the pointer to
// that file. Addr contains an address of the message and the file within the
// message slice to update.  It will return an error if the address references
// out of range.
func Update(msgs []types.Message, addr Addr, fn func(*slack.File) error) error {
	if addr.idxParMsg != Root {
		return Update(
			msgs[addr.idxParMsg].ThreadReplies,
			Addr{idxMsg: addr.idxMsg, idxParMsg: Root, idxFile: addr.idxFile},
			fn,
		)
	}

	if addr.idxMsg < 0 || len(msgs) <= addr.idxMsg {
		return errors.New("invalid message reference")
	}
	if addr.idxFile < 0 || len(msgs[addr.idxMsg].Files) < addr.idxFile {
		return errors.New("invalid file reference")
	}
	if err := fn(&msgs[addr.idxMsg].Files[addr.idxFile]); err != nil {
		return err
	}
	return nil
}

// Extract scans the message slice msgs, and calls fn for each file it
// finds. fn is called with the copy of the file and the files' address in the
// provided message slice.  idxParentMsg is the index of the parent message (for
// message replies slice), or refRoot if it's the topmost messages slice (see
// invocation in downloadFn).
func Extract(msgs []types.Message, idxParentMsg int, fn func(file slack.File, addr Addr) error) error {
	if fn == nil {
		return errors.New("extractFiles: internal error: no callback function")
	}
	for iMsg := range msgs {
		if len(msgs[iMsg].Files) > 0 {
			for fileIdx, file := range msgs[iMsg].Files {
				if err := fn(file, Addr{idxMsg: iMsg, idxParMsg: idxParentMsg, idxFile: fileIdx}); err != nil {
					return err
				}
			}
		}
		if len(msgs[iMsg].ThreadReplies) > 0 {
			if err := Extract(msgs[iMsg].ThreadReplies, iMsg, fn); err != nil {
				return err
			}
		}
	}
	return nil
}
