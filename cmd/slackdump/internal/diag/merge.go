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
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/internal/chunk"
	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase"
	"github.com/rusq/slackdump/v4/internal/convert"
	"github.com/rusq/slackdump/v4/source"
)

var cmdMerge = &base.Command{
	Run:       runMerge,
	UsageLine: "slackdump tools merge [flags] <target database> <source1> [source2 ... ]",
	Short:     "merges sources from the same workspace into an existing database archive",
	Long: `# Command Merge
Allows to merge different Slackdump sources into an existing database archive.

Make a backup of <database archive> before running this tool.

Merge does not perform deduplication.  If the input archives may overlap,
it is recommended to run ` + "`slackdump tools dedupe`" + ` on the merged
archive afterwards.

Limitations:
- Target database must exist;
- All sources must be from the same workspace.
`,
	FlagMask:   cfg.OmitAll,
	PrintFlags: true,
	Commands:   []*base.Command{},
	HideWizard: true,
}

type mergeParams struct {
	WithFiles   bool
	WithAvatars bool
}

var (
	checkOnly  bool
	mergeFlags = mergeParams{
		WithFiles:   true,
		WithAvatars: true,
	}
)

func init() {
	cmdMerge.Flag.BoolVar(&checkOnly, "check", false, "checks if the archives are mergeable, doesn't run the merge")
	cmdMerge.Flag.BoolVar(&mergeFlags.WithFiles, "files", true, "copy file attachments from sources")
	cmdMerge.Flag.BoolVar(&mergeFlags.WithAvatars, "avatars", true, "copy user avatars from sources")
}

func runMerge(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) < 2 {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("expected target database and at least one source")
	}
	targetPath := args[0]
	sourcePaths := args[1:]

	if err := verifyWorkspaces(ctx, targetPath, sourcePaths); err != nil {
		base.SetExitStatus(base.SInvalidParameters)
		return err
	}

	if checkOnly {
		fmt.Println("archives are compatible")
		return nil
	}

	target, err := resolveMergeTarget(targetPath)
	if err != nil {
		base.SetExitStatus(base.SInvalidParameters)
		return err
	}

	conn, srcs, err := open(ctx, target.Path, sourcePaths)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	defer conn.Close()
	defer closeSources(srcs)

	for i, src := range srcs {
		slog.InfoContext(ctx, "merging source", "n", i+1, "total", len(srcs), "name", src.Name())
		if err := mergeSource(ctx, target, conn, src, mergeFlags.WithFiles, mergeFlags.WithAvatars); err != nil {
			base.SetExitStatus(base.SApplicationError)
			return fmt.Errorf("merging source %q: %w", sourcePaths[i], err)
		}
	}
	return nil
}

type mergeTarget struct {
	Path       string
	ArchiveDir string
}

func resolveMergeTarget(targetPath string) (mergeTarget, error) {
	fi, err := os.Stat(targetPath)
	if err != nil {
		return mergeTarget{}, err
	}
	if fi.IsDir() {
		return mergeTarget{Path: targetPath, ArchiveDir: targetPath}, nil
	}
	return mergeTarget{Path: targetPath, ArchiveDir: filepath.Dir(targetPath)}, nil
}

// verifyWorkspaces opens target and all sources to verify they belong to the
// same Slack workspace.  All handles are closed before returning.
func verifyWorkspaces(ctx context.Context, targetPath string, sourcePaths []string) error {
	targetSrc, err := source.Load(ctx, targetPath)
	if err != nil {
		return fmt.Errorf("opening target: %w", err)
	}
	defer targetSrc.Close()

	var checkSrcs []source.Sourcer
	defer func() { closeSources(checkSrcs) }()

	for _, p := range sourcePaths {
		src, err := source.Load(ctx, p)
		if err != nil {
			return fmt.Errorf("opening source %q: %w", p, err)
		}
		checkSrcs = append(checkSrcs, src)
	}

	return checkArchives(ctx, targetSrc, checkSrcs...)
}

// open validates the target as a database archive, opens a RW connection to
// it, and opens each source path.
func open(ctx context.Context, targetPath string, sourcePaths []string) (*sqlx.DB, []source.Sourcer, error) {
	conn, err := ensureDb(ctx, targetPath)
	if err != nil {
		return nil, nil, err
	}
	srcs := make([]source.Sourcer, 0, len(sourcePaths))
	for _, p := range sourcePaths {
		src, err := source.Load(ctx, p)
		if err != nil {
			conn.Close()
			closeSources(srcs)
			return nil, nil, fmt.Errorf("opening %q: %w", p, err)
		}
		srcs = append(srcs, src)
	}
	return conn, srcs, nil
}

// checkArchives verifies that all sources belong to the same workspace as the
// target.  If a workspace ID cannot be determined for a source, a warning is
// logged and the source is skipped.
func checkArchives(ctx context.Context, target source.SourceResumeCloser, sources ...source.Sourcer) error {
	targetTeamID, err := getTeamID(ctx, target)
	if err != nil {
		if errors.Is(err, source.ErrNotFound) {
			slog.WarnContext(ctx, "cannot determine target workspace, skipping compatibility check")
			return nil
		}
		return fmt.Errorf("target workspace: %w", err)
	}

	for i, src := range sources {
		teamID, err := getTeamID(ctx, src)
		if err != nil {
			if errors.Is(err, source.ErrNotFound) {
				slog.WarnContext(ctx, "cannot determine source workspace, skipping check", "source", src.Name())
				continue
			}
			return fmt.Errorf("source[%d] %q workspace: %w", i, src.Name(), err)
		}
		if teamID != targetTeamID {
			return fmt.Errorf("source[%d] %q is from workspace %q, target is from workspace %q", i, src.Name(), teamID, targetTeamID)
		}
	}
	return nil
}

// getTeamID returns the Slack team ID from the source using a fallback chain:
// WorkspaceInfo → majority team among users → first public channel's team.
func getTeamID(ctx context.Context, src source.Sourcer) (string, error) {
	wsi, err := src.WorkspaceInfo(ctx)
	if err == nil {
		return wsi.TeamID, nil
	}
	if !errors.Is(err, source.ErrNotFound) && !errors.Is(err, source.ErrNotSupported) {
		return "", fmt.Errorf("workspace info: %w", err)
	}

	teamID, err := usersTeamID(ctx, src)
	if err == nil {
		return teamID, nil
	}
	return channelsTeamID(ctx, src)
}

// usersTeamID returns the team ID that the majority of users in src belong to.
func usersTeamID(ctx context.Context, src source.Sourcer) (string, error) {
	users, err := src.Users(ctx)
	if err != nil {
		return "", fmt.Errorf("getting users: %w", err)
	}
	if len(users) == 0 {
		return "", errors.New("no users found")
	}
	teams := make(map[string]int, 1)
	for _, u := range users {
		teams[u.TeamID]++
	}
	var maxUsers int
	var teamID string
	for t, c := range teams {
		if c > maxUsers {
			maxUsers = c
			teamID = t
		}
	}
	if maxUsers == 0 {
		return "", source.ErrNotFound
	}
	maxCount := 0
	for _, v := range teams {
		if v == maxUsers {
			maxCount++
		}
	}
	if maxCount > 1 {
		return "", errors.New("ambiguous team count")
	}
	return teamID, nil
}

// channelsTeamID returns the team ID from the first public channel with a
// known shared team ID.
func channelsTeamID(ctx context.Context, src source.Sourcer) (string, error) {
	channels, err := src.Channels(ctx)
	if err != nil {
		return "", fmt.Errorf("getting channels: %w", err)
	}
	for _, c := range channels {
		if !c.IsGroup && !c.IsIM && !c.IsMpIM && len(c.SharedTeamIDs) > 0 && c.SharedTeamIDs[0] != "" {
			return c.SharedTeamIDs[0], nil
		}
	}
	return "", source.ErrNotFound
}

// mergeSource copies all data from src into a new session in the target
// database.  Files and avatars are copied when the corresponding flags are
// true and the source has them.
func mergeSource(ctx context.Context, target mergeTarget, conn *sqlx.DB, src source.Sourcer, withFiles, withAvatars bool) error {
	dbp, err := dbase.New(ctx, conn, bootstrap.SessionInfo("merge"), dbase.WithOnlyNewOrChangedUsers(true))
	if err != nil {
		return fmt.Errorf("creating session: %w", err)
	}
	defer dbp.Close()

	// Workspace info
	if wsi, err := src.WorkspaceInfo(ctx); err == nil {
		if encErr := dbp.Encode(ctx, &chunk.Chunk{
			Type:          chunk.CWorkspaceInfo,
			WorkspaceInfo: wsi,
		}); encErr != nil {
			slog.WarnContext(ctx, "encoding workspace info", "error", encErr)
		}
	} else if !errors.Is(err, source.ErrNotFound) && !errors.Is(err, source.ErrNotSupported) {
		return fmt.Errorf("getting workspace info: %w", err)
	}

	// Users
	users, err := src.Users(ctx)
	if err != nil && !errors.Is(err, source.ErrNotFound) && !errors.Is(err, source.ErrNotSupported) {
		return fmt.Errorf("getting users: %w", err)
	}
	if len(users) > 0 {
		if err := dbp.Encode(ctx, &chunk.Chunk{
			Type:   chunk.CUsers,
			Users:  users,
			IsLast: true,
			Count:  int32(len(users)),
		}); err != nil {
			return fmt.Errorf("encoding users: %w", err)
		}
	}

	// Channels
	channels, err := src.Channels(ctx)
	if err != nil {
		return fmt.Errorf("getting channels: %w", err)
	}
	if err := dbp.Encode(ctx, &chunk.Chunk{
		Type:     chunk.CChannels,
		Channels: channels,
		IsLast:   true,
		Count:    int32(len(channels)),
	}); err != nil {
		return fmt.Errorf("encoding channels: %w", err)
	}

	// Setup file copier
	var (
		fc     *convert.FileCopier
		trgFSA fsadapter.FS
	)
	if withFiles || withAvatars {
		trgFSA = fsadapter.NewDirectory(target.ArchiveDir)
	}
	if withFiles {
		fc = convert.NewFileCopier(src, trgFSA, source.MattermostFilepath, true)
	}

	// Per-channel data
	for i := range channels {
		ch := &channels[i]

		// Channel info
		if ci, err := src.ChannelInfo(ctx, ch.ID); err == nil {
			if encErr := dbp.Encode(ctx, &chunk.Chunk{
				Type:      chunk.CChannelInfo,
				ChannelID: ch.ID,
				Channel:   ci,
			}); encErr != nil {
				slog.WarnContext(ctx, "encoding channel info", "channel", ch.ID, "error", encErr)
			}
		}

		// Messages
		msgIter, err := src.AllMessages(ctx, ch.ID)
		if err != nil {
			slog.WarnContext(ctx, "getting messages", "channel", ch.ID, "error", err)
			continue
		}
		var msgs []slack.Message
		for msg, err := range msgIter {
			if err != nil {
				slog.WarnContext(ctx, "iterating messages", "channel", ch.ID, "error", err)
				break
			}
			msgs = append(msgs, msg)
		}
		if len(msgs) == 0 {
			continue
		}
		if err := dbp.Encode(ctx, &chunk.Chunk{
			Type:      chunk.CMessages,
			ChannelID: ch.ID,
			Messages:  msgs,
			IsLast:    true,
			Count:     int32(len(msgs)),
		}); err != nil {
			return fmt.Errorf("encoding messages for %s: %w", ch.ID, err)
		}

		// File attachments
		if withFiles && src.Files().Type() != source.STnone {
			for j := range msgs {
				if err := fc.Copy(ch, &msgs[j]); err != nil {
					slog.WarnContext(ctx, "copying files", "channel", ch.ID, "ts", msgs[j].Timestamp, "error", err)
				}
			}
		}

		// Thread messages
		for j := range msgs {
			if msgs[j].ReplyCount == 0 {
				continue
			}
			threadIter, err := src.AllThreadMessages(ctx, ch.ID, msgs[j].Timestamp)
			if err != nil {
				slog.WarnContext(ctx, "getting thread", "channel", ch.ID, "ts", msgs[j].Timestamp, "error", err)
				continue
			}
			var threadMsgs []slack.Message
			for tm, err := range threadIter {
				if err != nil {
					slog.WarnContext(ctx, "iterating thread", "channel", ch.ID, "ts", msgs[j].Timestamp, "error", err)
					break
				}
				threadMsgs = append(threadMsgs, tm)
			}
			if len(threadMsgs) == 0 {
				continue
			}
			if err := dbp.Encode(ctx, &chunk.Chunk{
				Type:      chunk.CThreadMessages,
				ChannelID: ch.ID,
				ThreadTS:  msgs[j].Timestamp,
				Messages:  threadMsgs,
				IsLast:    true,
				Parent:    &msgs[j],
				Count:     int32(len(threadMsgs)),
			}); err != nil {
				return fmt.Errorf("encoding thread for %s/%s: %w", ch.ID, msgs[j].Timestamp, err)
			}
			if withFiles && src.Files().Type() != source.STnone {
				for k := range threadMsgs {
					if err := fc.Copy(ch, &threadMsgs[k]); err != nil {
						slog.WarnContext(ctx, "copying thread files", "channel", ch.ID, "thread_ts", msgs[j].Timestamp, "ts", threadMsgs[k].Timestamp, "error", err)
					}
				}
			}
		}
	}

	// Avatars
	if withAvatars && len(users) > 0 && src.Avatars().Type() != source.STnone {
		copyAvatars(ctx, src.Avatars(), users, trgFSA)
	}

	return nil
}

// copyAvatars copies avatars for all users from avst into trgFSA.  Errors are
// logged as warnings and do not abort the operation.
func copyAvatars(ctx context.Context, avst source.Storage, users []slack.User, trgFSA fsadapter.FS) {
	for _, u := range users {
		srcLoc, err := avst.File(source.AvatarParams(&u))
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				slog.WarnContext(ctx, "locating avatar", "user", u.ID, "error", err)
			}
			continue
		}
		dstLoc := path.Join(chunk.AvatarsDir, path.Join(source.AvatarParams(&u)))
		srcFile, err := avst.FS().Open(srcLoc)
		if err != nil {
			slog.WarnContext(ctx, "opening avatar", "user", u.ID, "error", err)
			continue
		}
		dstFile, err := trgFSA.Create(dstLoc)
		if err != nil {
			srcFile.Close()
			slog.WarnContext(ctx, "creating avatar destination", "user", u.ID, "error", err)
			continue
		}
		if _, err := io.Copy(dstFile, srcFile); err != nil {
			slog.WarnContext(ctx, "copying avatar", "user", u.ID, "error", err)
		}
		srcFile.Close()
		dstFile.Close()
	}
}

// closeSources closes any source that implements io.Closer, logging errors as
// warnings.
func closeSources(srcs []source.Sourcer) {
	for _, src := range srcs {
		if c, ok := src.(io.Closer); ok {
			if err := c.Close(); err != nil {
				slog.Warn("closing source", "name", src.Name(), "error", err)
			}
		}
	}
}
