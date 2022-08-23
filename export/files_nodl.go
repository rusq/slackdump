package export

import (
	"context"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/internal/structures/files"
	"github.com/rusq/slackdump/v2/types"
	"github.com/slack-go/slack"
)

// fileUpdater does not download any files, it just updates the link adding
// a token query parameter, if the token is set.
type fileUpdater struct {
	baseDownloader
}

// Start does nothing.
func (fileUpdater) Start(context.Context) {}

// Stop does nothing.
func (fileUpdater) Stop() {}

// newFileUpdater returns an fileExporter that does not download any files,
// but updates the link adding a token query parameter, if the token is set.
func newFileUpdater(token string) fileUpdater {
	return fileUpdater{baseDownloader: baseDownloader{
		token: token,
	}}
}

// ProcessFunc returns the [slackdump.ProcessFunc] that updates the file link
// adding a token query parameter.
func (u fileUpdater) ProcessFunc(_ string) slackdump.ProcessFunc {
	if u.token == "" {
		return func(msg []types.Message, channelID string) (slackdump.ProcessResult, error) {
			return slackdump.ProcessResult{}, nil
		}
	}
	return func(msgs []types.Message, channelID string) (slackdump.ProcessResult, error) {
		total := 0
		if err := files.Extract(msgs, files.Root, func(file slack.File, addr files.Addr) error {
			return files.Update(msgs, addr, updateTokenFn(u.token))
		}); err != nil {
			return slackdump.ProcessResult{}, err
		}
		return slackdump.ProcessResult{Entity: entFiles, Count: total}, nil
	}
}
