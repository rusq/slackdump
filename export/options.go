package export

import (
	"log/slog"
	"time"

	"github.com/rusq/slackdump/v3/internal/structures"
)

// Config allows to configure slack export options.
type Config struct {
	Oldest      time.Time
	Latest      time.Time
	Logger      *slog.Logger
	List        *structures.EntityList
	Type        ExportType
	MemberOnly  bool
	ExportToken string
}

func (opt Config) IsFilesEnabled() bool {
	return opt.Type > TNoDownload
}
