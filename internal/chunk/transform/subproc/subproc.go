package subproc

import (
	"github.com/rusq/slackdump/v2/internal/structures/files"
	"github.com/slack-go/slack"
)

// Downloader is the interface that wraps the Download method.
type Downloader interface {
	// Download should download the file at the given URL and save it to the
	// given path.
	Download(fullpath string, url string) error
}

type baseSubproc struct {
	dcl Downloader
}

// ExportTokenUpdateFn returns a function that appends the token to every file
// URL in the given message.
func ExportTokenUpdateFn(token string) func(msg *slack.Message) error {
	fn := files.UpdateTokenFn(token)
	return func(msg *slack.Message) error {
		for i := range msg.Files {
			if err := fn(&msg.Files[i]); err != nil {
				return err
			}
		}
		return nil
	}
}

// isDownloadable returns true if the file can be downloaded.
func isDownloadable(f *slack.File) bool {
	return f.Mode != "hidden_by_limit" && f.Mode != "external" && !f.IsExternal
}
