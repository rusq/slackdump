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

var (
	auto bool
)

var cmdUpdate = &base.Command{
	Run:       runUpdate,
	UsageLine: "slackdump tools update [flags]",
	Short:     "Update slackdump to latest version (EXPERIMENTAL)",
	Long: `
# Update slackdump

Check for updates and optionally update slackdump to the latest version.

## Usage

Without flags, this command checks if a new version is available:

    slackdump tools update

To automatically update to the latest version, use the -auto flag:

    slackdump tools update -auto

## Supported Update Methods

The update mechanism is platform-aware and uses the appropriate method:

- **macOS with Homebrew**: If slackdump is installed via brew, it uses 
  'brew update && brew upgrade slackdump'. It also checks if Homebrew has
  the latest version and warns if the formula is behind the GitHub release.

- **Arch Linux**: Uses 'sudo pacman -Sy --noconfirm slackdump'. 
  Note: The --noconfirm flag auto-approves the installation without prompting.

- **Debian/Ubuntu**: Uses 'sudo apt update && sudo apt install --only-upgrade slackdump'.
  APT will still prompt for confirmation interactively.

- **Windows / Generic Binary**: Downloads the latest binary from GitHub releases
  and replaces the current executable. A backup of the current binary is created
  before replacement.

## Security Considerations

When using -auto with package managers:
- Pacman (Arch Linux) uses --noconfirm which skips confirmation prompts
- APT (Debian/Ubuntu) may prompt for confirmation via sudo
- Commands requiring sudo will display the exact command before execution
- Only the specific slackdump package is updated, not a full system upgrade

## Notes

- The update command requires internet connectivity
- For package manager updates (brew, pacman, apt), you may need sudo privileges
- Homebrew formulae may lag behind GitHub releases by a few hours/days
- The -auto flag is EXPERIMENTAL and should be used with caution
`,
	Flag:        flag.FlagSet{},
	CustomFlags: false,
	FlagMask:    cfg.OmitAll,
	PrintFlags:  true,
	RequireAuth: false,
	HideWizard:  false,
}

func init() {
	cmdUpdate.Flag.BoolVar(&auto, "auto", false, "automatically update to the latest version")
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

	if curr.Equal(latest) {
		slog.InfoContext(ctx, "You are running the latest version")
		return nil
	}

	slog.WarnContext(ctx, "A new version is available", "version", latest.Version)

	if auto {
		slog.InfoContext(ctx, "Auto-update enabled, attempting to update...")
		if err := u.AutoUpdate(ctx, latest); err != nil {
			return err
		}
		slog.InfoContext(ctx, "Update completed successfully", "version", latest.Version)
	}

	return nil
}
