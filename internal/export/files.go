// Package files contains some additional file logic.
package export

import (
	"errors"

	"github.com/rusq/slackdump/v2/types"
	"github.com/slack-go/slack"
)

// Root is the parent address for the topmost message chunk.
const Root = -1

// Addr is the address of the file in the messages slice.
//   idxMsg    - index of the message or message reply in the provided slice
//   idxParMsg - index of the parent message. If it is not equal to Root,
//               then it's the address of the message:
//
//                   msg[idxMsg].File[idxFile]
//
//               if it is not equal to Root, then it is assumed that it is
//               a reference to a message reply:
//
//                   msg[idxParMsg].ThreadReplies[idxMsg].File[idxFile]
//
//   idxFile   - index of the file in the message's file slice.
//
type Addr struct {
	idxMsg    int // index of the message in the messages slice
	idxParMsg int // index of the parent message, or refRoot if it's the address of the top level message
	idxFile   int // index of the file in the file slice.
}

// UpdateURLs updates the URL link for the files in message chunk msgs.  Addr
// contains an address of the message and the file within the message slice to
// update the URL for, and filename is the path to the file on the local
// filesystem.  It will return an error if the address references out of range.
func UpdateURLs(msgs []types.Message, addr Addr, filename string) error {
	if addr.idxParMsg != Root {
		return UpdateURLs(
			msgs[addr.idxParMsg].ThreadReplies,
			Addr{idxMsg: addr.idxMsg, idxParMsg: Root, idxFile: addr.idxFile},
			filename,
		)
	}

	if addr.idxMsg < 0 || len(msgs) <= addr.idxMsg {
		return errors.New("invalid message reference")
	}
	if addr.idxFile < 0 || len(msgs[addr.idxMsg].Files) < addr.idxFile {
		return errors.New("invalid file reference")
	}
	msgs[addr.idxMsg].Files[addr.idxFile].URLPrivateDownload = filename
	msgs[addr.idxMsg].Files[addr.idxFile].URLPrivate = filename
	return nil
}

// Extract scans the message slice msgs, and calls fn for each file it
// finds. fn is called with the copy of the file and that file's address in the
// provided message slice.  idxParentMsg is the index of the parent message (for
// message replies slice), or refRoot if it's the topmost messages slice (see
// invocation in downloadFn).
func Extract(msgs []types.Message, idxParentMsg int, fn func(file slack.File, addr Addr) error) error {
	if fn == nil {
		return errors.New("extractFiles: internal error: no callback function")
	}
	for mi := range msgs {
		if len(msgs[mi].Files) > 0 {
			for fileIdx, file := range msgs[mi].Files {
				if err := fn(file, Addr{idxMsg: mi, idxParMsg: idxParentMsg, idxFile: fileIdx}); err != nil {
					return err
				}
			}
		}
		if len(msgs[mi].ThreadReplies) > 0 {
			if err := Extract(msgs[mi].ThreadReplies, mi, fn); err != nil {
				return err
			}
		}
	}
	return nil
}
