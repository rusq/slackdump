package resume

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/bubbles/filemgr"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/cfgui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/dumpui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/updaters"
	"github.com/rusq/slackdump/v3/internal/source"
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
			},
		},
		{
			Name: "Optional parameters",
			Params: []cfgui.Parameter{
				cfgui.OnlyChannelUsers(),
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
		Name:        "Include threads",
		Value:       cfgui.Checkbox(resumeFlags.IncludeThreads),
		Description: "Scan existing threads (SLOW).",
		Updater:     updaters.NewBool(&resumeFlags.IncludeThreads),
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
