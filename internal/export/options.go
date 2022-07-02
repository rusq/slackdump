package export

import (
	"time"

	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/logger"
)

// Options allows to configure slack export options.
type Options struct {
	Oldest       time.Time
	Latest       time.Time
	IncludeFiles bool
	Logger       logger.Interface
	List         *structures.EntityList
}
