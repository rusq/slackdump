package export

import (
	"github.com/rusq/slack"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v3/internal/structures/files/dl"
	"github.com/rusq/slackdump/v3/logger"
)

// newV2FileExporter returns the appropriate exporter for the ExportType.
func newV2FileExporter(t ExportType, fs fsadapter.FS, cl *slack.Client, l logger.Interface, token string) dl.Exporter {
	switch t {
	default:
		l.Printf("unknown export type %s, not downloading any files", t)
		fallthrough
	case TNoDownload:
		return dl.NewFileUpdater(token)
	case TStandard:
		return dl.NewStd(fs, cl, l, token)
	case TMattermost:
		return dl.NewMattermost(fs, cl, l, token)
	}
}
