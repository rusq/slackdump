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
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sosodev/duration"

	"github.com/rusq/slackdump/v4"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/archive"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	dedupecmd "github.com/rusq/slackdump/v4/cmd/slackdump/internal/diag/dedupe"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase"
	"github.com/rusq/slackdump/v4/internal/chunk/control"
	"github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/source"
	"github.com/rusq/slackdump/v4/stream"
)

//go:embed assets/resume.md
var mdResume string

var CmdResume = &base.Command{
	UsageLine:   "slackdump resume [flags] <archive or directory> [link1 [link2 ...]]",
	Short:       "resumes archive process from the last checkpoint",
	Long:        mdResume,
	PrintFlags:  true,
	RequireAuth: true,
	FlagMask:    cfg.OmitOutputFlag | cfg.OmitUserCacheFlag | cfg.OmitChunkFileMode | cfg.OmitRecordFilesFlag | cfg.OmitChunkCacheFlag | cfg.OmitYesManFlag,
	Wizard:      archiveWizard,
}

type ResumeParams struct {
	// Refresh the list of channels from the server.  Allows
	// adding non-existing channels that appeared since the last
	// run.
	Refresh bool
	// IncludeThreads includes scanning of the threads in the archive
	// and checking if there are any new messages in them.
	IncludeThreads bool
	// RecordOnlyNewUsers if set to false (default), records only updated or
	// new users. If set to true, records all users from the workspace again,
	// not just changed.
	RecordOnlyNewUsers bool
	// Lookback specifies the lookback parameter. The "oldest" timestamp for
	// API requests will be set to Now()-Lookback.  This is required to capture
	// new threads on historical messages, otherwise, new threads on old messages
	// will not be fetched.
	Lookback *extDuration
	// SkipCompleteThreads skips threads where the database already holds all
	// replies (DB count == API reply_count + 1).  Faster, but won't detect
	// edited or deleted messages.  Use only when threads are append-only.
	SkipCompleteThreads bool
	// SkipStaleThreads, if set to a non-zero duration, drops thread entities
	// whose latest known reply is older than the duration before they are
	// dispatched.  This is a pre-API filter that avoids fetching the first
	// page of replies for dormant threads.  Default (zero/unset) = disabled.
	SkipStaleThreads *extDuration
	// SkipStaleChannels, if set to a non-zero duration, drops channel
	// entities whose latest known message is older than the duration before
	// they are dispatched.  Pair with a periodic full-sweep run so dormant
	// channels are still revisited for resurrection coverage.  Default
	// (zero/unset) = disabled.
	SkipStaleChannels *extDuration
	// Dedupe runs duplicate entity cleanup after a successful resume.
	Dedupe bool
}

var resumeFlags = ResumeParams{
	Lookback:          (*extDuration)(duration.FromTimeDuration(7 * 24 * time.Hour)),
	SkipStaleThreads:  new(extDuration),
	SkipStaleChannels: new(extDuration),
}

func init() {
	CmdResume.Run = runResume
	CmdResume.Flag.BoolVar(&resumeFlags.Refresh, "refresh", false, "refresh the list of channels")
	CmdResume.Flag.BoolVar(&resumeFlags.IncludeThreads, "threads", false, "include threads (slow, and flaky business)")
	CmdResume.Flag.BoolVar(&resumeFlags.RecordOnlyNewUsers, "only-new-users", true, "record only new or updated users")
	CmdResume.Flag.Var(resumeFlags.Lookback, "lookback", "lookback window `duration`")
	CmdResume.Flag.BoolVar(&resumeFlags.SkipCompleteThreads, "skip-complete-threads", false, "skip threads where DB already has all replies (faster, but won't detect edits/deletes)")
	CmdResume.Flag.Var(resumeFlags.SkipStaleThreads, "skip-stale-threads", "skip thread entities whose latest reply is older than this `duration` (default: disabled)")
	CmdResume.Flag.Var(resumeFlags.SkipStaleChannels, "skip-stale-channels", "skip channel entities whose latest message is older than this `duration` (default: disabled; pair with a periodic full-sweep run)")
	CmdResume.Flag.BoolVar(&resumeFlags.Dedupe, "dedupe", false, "run dedupe cleanup after successful resume finish")
}

var runDedupe = func(ctx context.Context, conn *sqlx.DB, opts dedupecmd.Options) (dedupecmd.Result, error) {
	return dedupecmd.Run(ctx, conn, opts)
}

var (
	errRunArchiveController    = errors.New("error running archive controller")
	errFinishArchiveController = errors.New("error finalizing archive controller")
)

type archiveRunner interface {
	RunNoTransform(ctx context.Context, latest *structures.EntityList) error
	Finish() error
}

func runResume(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) < 1 {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("expected at least one argument")
	}
	dir := args[0]

	// parse the entity list, if it's present.
	list, err := structures.NewEntityList(args[1:])
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return err
	}

	src, err := source.Load(ctx, dir)
	if err != nil {
		base.SetExitStatus(base.SInvalidParameters)
		return err
	}
	defer src.Close() // ensure the source is closed in case we return early.

	if !src.Type().Has(source.FDatabase) {
		base.SetExitStatus(base.SInvalidParameters)
		return fmt.Errorf("source type %q does not support resume, use 'slackdump convert -f database' to convert it", src.Type())
	}

	threadCutoff := computeCutoff(resumeFlags.SkipStaleThreads)
	channelCutoff := computeCutoff(resumeFlags.SkipStaleChannels)
	latestResult, err := latest(ctx, src, resumeFlags.IncludeThreads, resumeFlags.SkipCompleteThreads, time.Duration((*duration.Duration)(resumeFlags.Lookback).ToTimeDuration()), threadCutoff, channelCutoff, list)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("error loading latest timestamps: %w", err)
	}
	switch decideResume(latestResult) {
	case resumeDecisionInvalidArchive:
		base.SetExitStatus(base.SInvalidParameters)
		return fmt.Errorf("the archive does not contain any data: %s", dir)
	case resumeDecisionNoop:
		cfg.Log.InfoContext(ctx, "all resume entities were skipped by stale filters", "database", dir, "skipped", latestResult.skippedStale)
		return nil
	}

	client, err := bootstrap.Slack(ctx)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return fmt.Errorf("error creating slackdump session: %w", err)
	}
	info, err := client.AuthTestContext(ctx)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return fmt.Errorf("error getting workspace info: %w", err)
	}

	// ensure the repository is for the same workspace.
	if err := ensureSameWorkspace(ctx, src, info); err != nil {
		base.SetExitStatus(base.SInitializationError)
		return fmt.Errorf("error ensuring the same workspace: %w", err)
	}

	// closing off the sourcer, as we don't need it anymore.
	if err := src.Close(); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("error closing source: %w", err)
	}

	// connecting to the database in read-write mode.
	wconn, err := bootstrap.Database(dir)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return fmt.Errorf("error opening database: %w", err)
	}
	defer wconn.Close()

	cf := control.Flags{
		Refresh:       resumeFlags.Refresh,
		ChannelUsers:  cfg.OnlyChannelUsers,
		ChannelTypes:  cfg.ChannelTypes,
		IncludeLabels: cfg.IncludeCustomLabels,
		MemberOnly:    cfg.MemberOnly,
	}
	// inclusive is false, because we don't want to include the latest message
	// which is already in the database.
	streamOpts := []stream.Option{stream.OptInclusive(false)}
	if resumeFlags.SkipCompleteThreads {
		streamOpts = append(streamOpts, stream.OptSkipThreadFunc(dbase.NewThreadSkipper(wconn)))
	}
	ctrl, err := archive.DBController(
		ctx,
		cmd.Name(),
		wconn,
		client,
		dir,
		cf,
		streamOpts,
		archive.WithFileDeduplication(),
		archive.WithDatabaseOptions(
			dbase.WithOnlyNewOrChangedUsers(resumeFlags.RecordOnlyNewUsers),
		),
	)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return fmt.Errorf("error creating archive controller: %w", err)
	}
	defer ctrl.Close()

	if err := runArchiveAndCleanup(ctx, ctrl, latestResult.list, wconn, dir, resumeFlags.Dedupe); err != nil {
		if errors.Is(err, errRunArchiveController) {
			base.SetExitStatus(base.SApplicationError)
		}
		if errors.Is(err, errFinishArchiveController) {
			base.SetExitStatus(base.SApplicationError)
		}
		return err
	}

	return nil
}

type resumeDecision int

const (
	resumeDecisionContinue resumeDecision = iota
	resumeDecisionInvalidArchive
	resumeDecisionNoop
)

func decideResume(r latestResult) resumeDecision {
	if r.list != nil && !r.list.IsEmpty() {
		return resumeDecisionContinue
	}
	if !r.hasSourceData {
		return resumeDecisionInvalidArchive
	}
	if r.skippedStale > 0 {
		return resumeDecisionNoop
	}
	return resumeDecisionInvalidArchive
}

func runArchiveAndCleanup(ctx context.Context, runner archiveRunner, latest *structures.EntityList, conn *sqlx.DB, dir string, dedupeEnabled bool) error {
	if err := runner.RunNoTransform(ctx, latest); err != nil {
		return fmt.Errorf("%w: %w", errRunArchiveController, err)
	}
	if err := runner.Finish(); err != nil {
		return fmt.Errorf("%w: %w", errFinishArchiveController, err)
	}
	if err := runDedupeAfterFinish(ctx, conn, dir, dedupeEnabled); err != nil {
		slog.WarnContext(ctx, "post-finish dedupe failed; resume run is complete", "database", dir, "error", err)
	}
	return nil
}

func runDedupeAfterFinish(ctx context.Context, conn *sqlx.DB, dir string, enabled bool) error {
	if !enabled {
		return nil
	}
	_, err := runDedupe(ctx, conn, dedupecmd.Options{
		Execute:  true,
		Database: dir,
	})
	return err
}

type latestResult struct {
	list          *structures.EntityList
	hasSourceData bool
	skippedStale  int
}

func latest(ctx context.Context, src source.Resumer, includeThreads bool, skipCompleteThreads bool, lookBack time.Duration, threadCutoff, channelCutoff *time.Time, other *structures.EntityList) (latestResult, error) {
	if lookBack > 0 {
		lookBack = -lookBack
	}
	latest, err := src.Latest(ctx)
	if err != nil {
		return latestResult{}, fmt.Errorf("error loading latest timestamps: %w", err)
	}
	result := latestResult{
		list:          &structures.EntityList{},
		hasSourceData: len(latest) > 0,
	}
	if len(latest) == 0 && (other == nil || other.IsEmpty()) {
		return result, nil
	}

	if cfg.Verbose {
		strlatest(latest)
	}

	ei := make([]structures.EntityItem, 0, len(latest))
	for sl, ts := range latest {
		if sl.IsThread() && !includeThreads {
			continue
		}
		if sl.IsThread() && threadCutoff != nil && ts.Before(*threadCutoff) {
			result.skippedStale++
			continue
		}
		if !sl.IsThread() && channelCutoff != nil && ts.Before(*channelCutoff) {
			result.skippedStale++
			continue
		}
		item := structures.EntityItem{
			Id:      sl.String(),
			Oldest:  ts.Add(lookBack),
			Latest:  time.Time(cfg.Latest),
			Include: true,
		}
		ei = append(ei, item)
		debugprint(fmt.Sprintf("%s: %d->%d", item.Id, ts.UTC().UnixMicro(), item.Oldest.UnixMicro()))
	}
	el := structures.NewEntityListFromItems(ei...)
	el.Overlay(other)
	result.list = el

	return result, nil
}

func debugprint(a ...any) {
	if cfg.Verbose {
		fmt.Println(a...)
	}
}

func strlatest(l map[structures.SlackLink]time.Time) string {
	var buf strings.Builder
	tw := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', 0)
	fmt.Fprintln(tw, "Group ID\tLatest")
	for gid, ts := range l {
		fmt.Fprintf(tw, "%s\t%s\n", gid, ts.Format("2006-01-02 15:04:05 MST"))
	}
	if err := tw.Flush(); err != nil {
		slog.Error("flushing went wrong", "error", err)
	}
	return buf.String()
}

func ensureSameWorkspace(ctx context.Context, src source.Sourcer, info *slackdump.WorkspaceInfo) error {
	lg := cfg.Log.With("in", "ensureSameWorkspace")
	var srcTeamID string
	wsp, err := src.WorkspaceInfo(ctx)
	if err != nil {
		if !errors.Is(err, source.ErrNotFound) {
			return fmt.Errorf("error getting workspace info: %w", err)
		}

		lg.DebugContext(ctx, "workspace info not found, trying to get team ID from users")
		srcTeamID, err = usersTeam(ctx, src)
		if err != nil {

			lg.DebugContext(ctx, "team ID not found in users, trying to get team ID from channels")
			srcTeamID, err = channelsTeam(ctx, src)
			if err != nil {
				lg.DebugContext(ctx, `¯\_(ツ)_/¯`)
				return source.ErrNotFound
			}
		}
	} else {
		srcTeamID = wsp.TeamID
	}

	if srcTeamID != info.TeamID {
		return fmt.Errorf("database workspace %s does not match session workspace %s", srcTeamID, info.TeamID)
	}
	return nil
}

// usersTeam returns the team ID of the team with the most users.
func usersTeam(ctx context.Context, src source.Sourcer) (string, error) {
	users, err := src.Users(ctx)
	if err != nil {
		return "", fmt.Errorf("error getting users: %w", err)
	}
	if len(users) == 0 {
		return "", errors.New("no users found")
	}

	// count users per team
	teams := make(map[string]int, 1)
	for _, u := range users {
		teams[u.TeamID]++
	}

	// find the team with most users
	var (
		maxUsers int
		teamID   string
	)
	for t, c := range teams {
		if c > maxUsers {
			maxUsers = c
			teamID = t
		}
	}
	if maxUsers == 0 {
		return "", source.ErrNotFound
	}

	// check if there are more than one team with max users
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

// channelsTeam returns the team ID of the first public channel with a
// shared team ID.
func channelsTeam(ctx context.Context, src source.Sourcer) (string, error) {
	channels, err := src.Channels(ctx)
	if err != nil {
		return "", fmt.Errorf("error getting channels: %w", err)
	}
	if len(channels) == 0 {
		return "", errors.New("no channels found")
	}
	for _, c := range channels {
		if !c.IsGroup && !c.IsIM && !c.IsMpIM && len(c.SharedTeamIDs) > 0 && c.SharedTeamIDs[0] != "" {
			return c.SharedTeamIDs[0], nil
		}
	}
	return "", source.ErrNotFound
}

// computeCutoff returns the absolute time before which entities are
// considered stale.  Returns nil when d is nil or represents a zero/
// negative duration, in which case callers should treat the filter as
// disabled.
func computeCutoff(d *extDuration) *time.Time {
	if d == nil {
		return nil
	}
	dur := time.Duration((*duration.Duration)(d).ToTimeDuration())
	if dur <= 0 {
		return nil
	}
	t := time.Now().Add(-dur)
	return &t
}

type extDuration duration.Duration

func (d *extDuration) Set(s string) error {
	// match ISO 8601 duration format
	s = strings.ToUpper(s)
	if !strings.HasPrefix(s, "P") {
		s = "P" + s
	}
	dur, err := duration.Parse(s)
	if err != nil {
		return err
	}
	*d = extDuration(*dur)
	return nil
}

func (d *extDuration) String() string {
	return strings.ToLower((*duration.Duration)(d).String())
}

func (d *extDuration) IsBoolFlag() bool {
	return false
}
