package export

import (
	"time"

	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/logger"
)

// Config allows to configure slack export options.
type Config struct {
	Oldest      time.Time
	Latest      time.Time
	Logger      logger.Interface
	List        *structures.EntityList
	Type        ExportType
	ExportToken string
}

func (opt Config) IsFilesEnabled() bool {
	return opt.Type > TNoDownload
}
