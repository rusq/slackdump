package dl

// no download, but update the token if required.

import (
	"context"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/internal/structures/files"
	"github.com/rusq/slackdump/v2/types"
)

// NoDownload does not download any files, it just updates the link adding
// a token query parameter, if the token is set.
type NoDownload struct {
	baseDownloader
}

// Start does nothing.
func (NoDownload) Start(context.Context) {}

// Stop does nothing.
func (NoDownload) Stop() {}

// NewFileUpdater returns an fileExporter that does not download any files,
// but updates the link adding a token query parameter, if the token is set.
func NewFileUpdater(token string) NoDownload {
	return NoDownload{baseDownloader: baseDownloader{
		token: token,
	}}
}

// ProcessFunc returns the [slackdump.ProcessFunc] that updates the file link
// adding a token query parameter.
func (u NoDownload) ProcessFunc(_ string) slackdump.ProcessFunc {
	if u.token == "" {
		// return dummy function, if the token is empty.
		return func(msg []types.Message, channelID string) (slackdump.ProcessResult, error) {
			return slackdump.ProcessResult{}, nil
		}
	}
	return func(msgs []types.Message, channelID string) (slackdump.ProcessResult, error) {
		total := 0
		if err := files.Extract(msgs, files.Root, func(file slack.File, addr files.Addr) error {
			return files.Update(msgs, addr, files.UpdateTokenFn(u.token))
		}); err != nil {
			return slackdump.ProcessResult{}, err
		}
		return slackdump.ProcessResult{Entity: entFiles, Count: total}, nil
	}
}
