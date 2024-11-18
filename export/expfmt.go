package export

import (
	"log/slog"

	"github.com/rusq/slack"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v3/internal/structures/files/dl"
)

// newV2FileExporter returns the appropriate exporter for the ExportType.
func newV2FileExporter(t ExportType, fs fsadapter.FS, cl *slack.Client, l *slog.Logger, token string) dl.Exporter {
	switch t {
	default:
		l.Warn("unknown export type, files won't be downloaded", "type", t)
		fallthrough
	case TNoDownload:
		return dl.NewFileUpdater(token)
	case TStandard:
		return dl.NewStd(fs, cl, l, token)
	case TMattermost:
		return dl.NewMattermost(fs, cl, l, token)
	}
}
