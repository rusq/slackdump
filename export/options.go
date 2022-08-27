package export

import (
	"time"

	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/logger"
)

// Options allows to configure slack export options.
type Options struct {
	Oldest      time.Time
	Latest      time.Time
	Logger      logger.Interface
	List        *structures.EntityList
	Type        ExportType
	ExportToken string
}

func (opt Options) IsFilesEnabled() bool {
	return opt.Type > TNoDownload
}
