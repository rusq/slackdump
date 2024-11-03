package cfgui

import (
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/updaters"
	"github.com/rusq/slackdump/v3/internal/structures"
)

// Common reusable parameters

func ChannelIDs(v *string, required bool) Parameter {
	name := "Channel IDs or URLs"
	descr := "List of channel IDs or URLs to dump"
	if required {
		name = "* " + name
		descr = descr + " (REQUIRED)"
	}
	return Parameter{
		Name:        name,
		Value:       *v,
		Description: descr,
		Inline:      true,
		Updater:     updaters.NewString(v, "", true, structures.ValidateEntityList),
	}
}
