package files

import (
	"context"

	"github.com/rusq/slackdump/v2"
)

//go:generate sh -c "mockgen -source files.go -destination interfaces_mock_test.go -package files"

// FileProcessor is the file exporter interface.
type FileProcessor interface {
	// ProcessFunc returns the process function that should be passed to
	// DumpMessagesRaw. It should be able to extract files from the messages
	// and download them.  If the dl is not started, i.e. if file
	// download is disabled, it should silently ignore the error and return
	// nil.
	ProcessFunc(channelName string) slackdump.ProcessFunc
}

type Exporter interface {
	FileProcessor
	StartStopper
}

type StartStopper interface {
	Start(ctx context.Context)
	Stop()
}
