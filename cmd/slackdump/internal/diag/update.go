package diag

import (
	"context"
	"errors"
	"flag"
	"log/slog"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/diag/updater"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
)

// In this file: Auto-update command

var cmdUpdate = &base.Command{
	Run:         runUpdate,
	UsageLine:   "slackdump tools update [flags]",
	Short:       "Update slackdump to latest version",
	Long:        "",
	Flag:        flag.FlagSet{},
	CustomFlags: false,
	FlagMask:    cfg.OmitAll,
	PrintFlags:  true,
	RequireAuth: false,
	HideWizard:  false,
}

func runUpdate(ctx context.Context, cmd *base.Command, args []string) error {
	// check the remote version
	u := updater.NewUpdater()
	curr, err := u.Current(ctx)
	if err != nil {
		if errors.Is(err, updater.ErrUnreleased) {
			slog.InfoContext(ctx, "You are running a development version, updates are disabled")
			return nil
		}
		return err
	}
	slog.DebugContext(ctx, "Current version", "version", curr.Version, "published_at", curr.PublishedAt)

	latest, err := u.Latest(ctx)
	if err != nil {
		return err
	}
	slog.DebugContext(ctx, "Latest version available", "version", latest.Version, "published_at", latest.PublishedAt)
	if eq, err := curr.Equal(latest); err != nil {
		return err
	} else {
		if eq {
			slog.InfoContext(ctx, "You are running the latest version")
		} else {
			slog.WarnContext(ctx, "A new version is available", "version", latest.Version)
		}
	}

	return nil
}
