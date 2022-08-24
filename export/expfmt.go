package export

import (
	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/rusq/slackdump/v2/internal/structures/files"
	"github.com/rusq/slackdump/v2/internal/structures/files/dl"
	"github.com/rusq/slackdump/v2/logger"
)

// newFileExporter returns the appropriate exporter for the ExportType.
func newFileExporter(t ExportType, fs fsadapter.FS, cl *slack.Client, l logger.Interface, token string) files.Exporter {
	switch t {
	default:
		l.Printf("unknown export type %s, not downloading any files", t)
		fallthrough
	case TNoDownload:
		return dl.NewFileUpdater(token)
	case TStandard:
		return dl.NewStdDl(fs, cl, l, token)
	case TMattermost:
		return dl.NewMattermostDl(fs, cl, l, token)
	}
}
