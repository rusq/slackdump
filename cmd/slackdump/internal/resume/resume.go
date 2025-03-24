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

	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/archive"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/chunk/control"
	"github.com/rusq/slackdump/v3/internal/source"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/stream"
)

//go:embed assets/resume.md
var mdResume string

var CmdResume = &base.Command{
	UsageLine:   "slackdump resume [flags] <archive or directory>",
	Short:       "resumes archive process from the last checkpoint",
	Long:        mdResume,
	PrintFlags:  true,
	RequireAuth: true,
	FlagMask:    cfg.OmitOutputFlag | cfg.OmitUserCacheFlag | cfg.OmitChunkFileMode | cfg.OmitRecordFilesFlag | cfg.OmitChunkCacheFlag,
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
}

var resumeFlags ResumeParams

func init() {
	CmdResume.Run = runResume
	CmdResume.Flag.BoolVar(&resumeFlags.Refresh, "refresh", false, "refresh the list of channels")
	CmdResume.Flag.BoolVar(&resumeFlags.IncludeThreads, "threads", false, "include threads (slow, and flaky business)")
}

func runResume(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) != 1 {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("expected exactly one argument")
	}
	dir := args[0]

	src, err := source.Load(ctx, dir)
	if err != nil {
		base.SetExitStatus(base.SInvalidParameters)
		return err
	}
	defer src.Close() // ensure the source is closed in case we return early.

	if !src.Type().Has(source.FDatabase) {
		base.SetExitStatus(base.SInvalidParameters)
		return fmt.Errorf("source type %q does not support resume, use slackdump convert to database format", src.Type())
	}

	latest, err := latest(ctx, src, resumeFlags.IncludeThreads)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("error loading latest timestamps: %w", err)
	}

	sess, err := bootstrap.SlackdumpSession(ctx)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return fmt.Errorf("error creating slackdump session: %w", err)
	}

	// ensure the repository is for the same workspace.
	if err := ensureSameWorkspace(ctx, src, sess.Info()); err != nil {
		base.SetExitStatus(base.SInitializationError)
		return fmt.Errorf("error ensuring the same workspace: %w", err)
	}

	// closing off the sourcer, as we don't need it anymore.
	if err := src.Close(); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("error closing source: %w", err)
	}

	// connecting to the database in read-write mode.
	wconn, _, err := bootstrap.Database(dir, cmd.Name())
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return fmt.Errorf("error opening database: %w", err)
	}
	defer wconn.Close()

	cf := control.Flags{
		Refresh:      resumeFlags.Refresh,
		ChannelUsers: cfg.OnlyChannelUsers,
	}
	// inclusive is false, because we don't want to include the latest message
	// which is already in the database.
	ctrl, err := archive.DBController(ctx, cmd, wconn, sess, dir, cf, stream.OptInclusive(false))
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return fmt.Errorf("error creating archive controller: %w", err)
	}
	defer ctrl.Close()

	if err := ctrl.Run(ctx, latest); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("error running archive controller: %w", err)
	}

	return nil
}

func latest(ctx context.Context, src source.Resumer, includeThreads bool) (*structures.EntityList, error) {
	latest, err := src.Latest(ctx)
	if err != nil {
		return nil, fmt.Errorf("error loading latest timestamps: %w", err)
	}
	if len(latest) == 0 {
		return &structures.EntityList{}, nil
	}

	if cfg.Verbose {
		strlatest(latest)
	}

	ei := make([]structures.EntityItem, 0, len(latest))
	for sl, ts := range latest {
		if sl.IsThread() && !includeThreads {
			continue
		}
		item := structures.EntityItem{
			Id:      sl.String(),
			Oldest:  ts,
			Latest:  time.Time(cfg.Latest),
			Include: true,
		}
		ei = append(ei, item)
		debugprint(fmt.Sprintf("%s: %d->%d", item.Id, ts.UTC().UnixMicro(), item.Oldest.UnixMicro()))
	}
	el := structures.NewEntityListFromItems(ei...)

	return el, nil
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
