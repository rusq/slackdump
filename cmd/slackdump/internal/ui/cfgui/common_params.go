package cfgui

import (
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/updaters"
	"github.com/rusq/slackdump/v3/internal/structures"
)

// Common reusable parameters

func ChannelIDs(v *string) Parameter {
	return Parameter{
		Name:        "* Channel IDs or URLs",
		Value:       *v,
		Description: "List of channel IDs or URLs to dump (REQUIRED)",
		Inline:      true,
		Updater:     updaters.NewString(v, "", true, structures.ValidateEntityList),
	}
}
