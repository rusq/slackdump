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
package resume

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/sosodev/duration"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui/bubbles/filemgr"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui/cfgui"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui/dumpui"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui/updaters"
	"github.com/rusq/slackdump/v4/source"
)

func archiveWizard(ctx context.Context, cmd *base.Command, args []string) error {
	w := &dumpui.Wizard{
		Title:       "Resume or Update Archive",
		Name:        "Resume",
		Cmd:         cmd,
		LocalConfig: configuration,
		// Help:        "Resume the archive process from the last checkpoint.",
		ValidateParamsFn: func() error {
			if dbfile == "" {
				return errors.New("no archive file selected")
			}
			return nil
		},
		ArgsFn: func() []string {
			return []string{filepath.Dir(dbfile)}
		},
	}
	return w.Run(ctx)
}

var dbfile string

func configuration() cfgui.Configuration {
	return cfgui.Configuration{
		{
			Name: "Refresh specific",
			Params: []cfgui.Parameter{
				cpArchivePicker(),
				cpRefresh(),
				cpIncludeThreads(),
				cpOnlyNewUsers(),
				cpLookback(),
			},
		},
		{
			Name: "Optional parameters",
			Params: []cfgui.Parameter{
				cfgui.OnlyChannelUsers(),
				cfgui.IncludeCustomLabels(),
				cfgui.Avatars(),
			},
		},
	}
}

func cpRefresh() cfgui.Parameter {
	return cfgui.Parameter{
		Name:        "Refresh the list of channels",
		Value:       cfgui.Checkbox(resumeFlags.Refresh),
		Description: "Include new channels that appeared since the last run.",
		Updater:     updaters.NewBool(&resumeFlags.Refresh),
	}
}

func cpIncludeThreads() cfgui.Parameter {
	return cfgui.Parameter{
		Name:        "Refresh threads",
		Value:       cfgui.Checkbox(resumeFlags.IncludeThreads),
		Description: "Scan existing threads (SLOW).",
		Updater:     updaters.NewBool(&resumeFlags.IncludeThreads),
	}
}

func cpOnlyNewUsers() cfgui.Parameter {
	return cfgui.Parameter{
		Name:        "Only New Or Changed Users",
		Value:       cfgui.Checkbox(resumeFlags.RecordOnlyNewUsers),
		Description: "Record only new or updated users (avoids user duplication).",
		Updater:     updaters.NewBool(&resumeFlags.RecordOnlyNewUsers),
	}
}

func cpArchivePicker() cfgui.Parameter {
	validator := func(s string) error {
		st, err := source.Type(s)
		if err != nil {
			return err
		}
		if !st.Has(source.FDatabase) {
			return errors.New("source type does not support resume")
		}
		return nil
	}
	model := filemgr.New(os.DirFS("."), ".", ".", 15, "*.sqlite")
	updater := updaters.NewFilepickModel(&dbfile, model, validator)
	param := cfgui.Parameter{
		Name:        "Archive to resume",
		Value:       dbfile,
		Description: "The directory or database file to resume.",
		Inline:      false,
		Updater:     updater,
	}
	return param
}

func cpLookback() cfgui.Parameter {
	return cfgui.Parameter{
		Name:        "Lookback",
		Value:       resumeFlags.Lookback.String(),
		Inline:      true,
		Description: "Duration to check for changed messages before the last message in the archive.",
		Updater:     updaters.NewISODuration((*duration.Duration)(resumeFlags.Lookback), false),
	}
}
