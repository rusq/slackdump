package dl

import (
	"context"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2"
)

//go:generate sh -c "mockgen -source files.go -destination interfaces_mock_test.go -package files"

// Exporter is the file exporter interface.
type Exporter interface {
	// ProcessFunc returns the process function that should be passed to
	// DumpMessagesRaw. It should be able to extract files from the messages
	// and download them.  If the dl is not started, i.e. if file
	// download is disabled, it should silently ignore the error and return
	// nil.
	ProcessFunc(channelName string) slackdump.ProcessFunc
	StartStopper
}

type StartStopper interface {
	Start(ctx context.Context)
	Stop()
}

// exportDownloader is the interface that dl.Client implements.  Used
// for mocking in tests.
type exportDownloader interface {
	DownloadFile(dir string, f slack.File) (string, error)
	StartStopper
}
