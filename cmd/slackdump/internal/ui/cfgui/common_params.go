// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
package cfgui

import (
	"github.com/charmbracelet/huh"

	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
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
		Updater:     updaters.NewString(v, "", false, structures.ValidateEntityList),
	}
}

// MemberOnly returns a checkbox parameter for Member Only flag.
func MemberOnly() Parameter {
	return Parameter{
		Name:        "Member Only",
		Value:       Checkbox(cfg.MemberOnly),
		Description: "Export only channels, which you belongs to.",
		Updater:     updaters.NewBool(&cfg.MemberOnly),
	}
}

// RecordFiles returns a checkbox parameter for Record Files flag.
func RecordFiles() Parameter {
	return Parameter{
		Name:        "Record Files",
		Value:       Checkbox(cfg.RecordFiles),
		Description: "Record file chunks in chunk files.",
		Updater:     updaters.NewBool(&cfg.RecordFiles),
	}
}

func Avatars() Parameter {
	return Parameter{
		Name:        "Download Avatars",
		Value:       Checkbox(cfg.WithAvatars),
		Description: "Download avatars.",
		Updater:     updaters.NewBool(&cfg.WithAvatars),
	}
}

func OnlyChannelUsers() Parameter {
	return Parameter{
		Name:        "Only Channel Users",
		Value:       Checkbox(cfg.OnlyChannelUsers),
		Description: "Only users participating in visible conversations are exported.",
		Updater:     updaters.NewBool(&cfg.OnlyChannelUsers),
	}
}

func ChannelTypes() Parameter {
	var items = map[string]struct {
		code        string
		description string
		selected    bool
	}{
		structures.CIM:      {code: structures.CIM, description: "Direct Messages"},
		structures.CMPIM:    {code: structures.CMPIM, description: "Group Messages"},
		structures.CPublic:  {code: structures.CPublic, description: "Public Messages"},
		structures.CPrivate: {code: structures.CPrivate, description: "Private Messages"},
	}

	for _, code := range cfg.ChannelTypes {
		v := items[code]
		v.selected = true
		items[code] = v
	}

	var options = make([]huh.Option[string], 0, len(slackdump.AllChanTypes))
	for _, code := range slackdump.AllChanTypes {
		item := items[code]
		options = append(options, huh.NewOption(item.description, item.code).Selected(item.selected))
	}

	return Parameter{
		Name:        "Channel Types",
		Value:       cfg.ChannelTypes.String(),
		Description: "Channel types to fetch",
		Updater: updaters.NewMultiSelect((*[]string)(&cfg.ChannelTypes), huh.NewMultiSelect[string]().
			Title("Choose Channel Types").
			Options(options...)),
	}
}

func IncludeCustomLabels() Parameter {
	return Parameter{
		Name:        "Include Custom Field Labels",
		Value:       Checkbox(cfg.IncludeCustomLabels),
		Description: "Channel users custom user profile fields labels (may result in request throttling).",
		Updater:     updaters.NewBool(&cfg.IncludeCustomLabels),
	}
}
