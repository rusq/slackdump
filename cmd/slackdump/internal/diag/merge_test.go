package diag

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v4/internal/chunk"
	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase"
	"github.com/rusq/slackdump/v4/source"
	"github.com/rusq/slackdump/v4/source/mock_source"
)

func TestResolveMergeTarget(t *testing.T) {
	t.Run("archive directory", func(t *testing.T) {
		archiveDir := newArchiveDir(t)

		got, err := resolveMergeTarget(archiveDir)
		require.NoError(t, err)
		require.Equal(t, mergeTarget{Path: archiveDir, ArchiveDir: archiveDir}, got)
	})

	t.Run("direct sqlite path", func(t *testing.T) {
		archiveDir := newArchiveDir(t)
		dbFile := filepath.Join(archiveDir, source.DefaultDBFile)

		got, err := resolveMergeTarget(dbFile)
		require.NoError(t, err)
		require.Equal(t, mergeTarget{Path: dbFile, ArchiveDir: archiveDir}, got)
	})

	t.Run("missing target path returns error", func(t *testing.T) {
		_, err := resolveMergeTarget(filepath.Join(t.TempDir(), "missing.sqlite"))
		require.Error(t, err)
	})
}

func TestVerifyWorkspaces(t *testing.T) {
	t.Run("matching workspaces succeeds", func(t *testing.T) {
		targetDir := newArchiveDir(t)
		writeMergeArchive(t, targetDir, []*chunk.Chunk{testWorkspaceChunk("T01")}, nil)

		sourceDir := newArchiveDir(t)
		writeMergeArchive(t, sourceDir, []*chunk.Chunk{testWorkspaceChunk("T01")}, nil)

		err := verifyWorkspaces(t.Context(), targetDir, []string{sourceDir})
		require.NoError(t, err)
	})

	t.Run("mismatched workspaces fails", func(t *testing.T) {
		targetDir := newArchiveDir(t)
		writeMergeArchive(t, targetDir, []*chunk.Chunk{testWorkspaceChunk("T01")}, nil)

		sourceDir := newArchiveDir(t)
		writeMergeArchive(t, sourceDir, []*chunk.Chunk{testWorkspaceChunk("T02")}, nil)

		err := verifyWorkspaces(t.Context(), targetDir, []string{sourceDir})
		require.Error(t, err)
		require.Contains(t, err.Error(), `workspace "T02"`)
	})

	t.Run("source open error is wrapped with source path", func(t *testing.T) {
		targetDir := newArchiveDir(t)
		writeMergeArchive(t, targetDir, []*chunk.Chunk{testWorkspaceChunk("T01")}, nil)

		missing := filepath.Join(t.TempDir(), "missing-source.sqlite")
		err := verifyWorkspaces(t.Context(), targetDir, []string{missing})
		require.Error(t, err)
		require.Contains(t, err.Error(), `opening source "`+missing+`"`)
	})
}

func TestCheckArchives(t *testing.T) {
	t.Run("all sources match target workspace", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		target := mock_source.NewMockSourceResumeCloser(ctrl)
		src := mock_source.NewMockSourcer(ctrl)

		target.EXPECT().WorkspaceInfo(gomock.Any()).Return(&slack.AuthTestResponse{TeamID: "T01"}, nil)
		src.EXPECT().WorkspaceInfo(gomock.Any()).Return(&slack.AuthTestResponse{TeamID: "T01"}, nil)

		err := checkArchives(t.Context(), target, src)
		require.NoError(t, err)
	})

	t.Run("source workspace mismatch returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		target := mock_source.NewMockSourceResumeCloser(ctrl)
		src := mock_source.NewMockSourcer(ctrl)
		src.EXPECT().Name().Return("src-1").AnyTimes()

		target.EXPECT().WorkspaceInfo(gomock.Any()).Return(&slack.AuthTestResponse{TeamID: "T01"}, nil)
		src.EXPECT().WorkspaceInfo(gomock.Any()).Return(&slack.AuthTestResponse{TeamID: "T02"}, nil)

		err := checkArchives(t.Context(), target, src)
		require.Error(t, err)
		require.Contains(t, err.Error(), `workspace "T02"`)
	})

	t.Run("target workspace not found skips compatibility check", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		target := mock_source.NewMockSourceResumeCloser(ctrl)

		target.EXPECT().WorkspaceInfo(gomock.Any()).Return(nil, source.ErrNotFound)
		target.EXPECT().Users(gomock.Any()).Return(nil, source.ErrNotFound)
		target.EXPECT().Channels(gomock.Any()).Return(nil, source.ErrNotFound)

		err := checkArchives(t.Context(), target)
		require.NoError(t, err)
	})

	t.Run("source workspace not found skips source", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		target := mock_source.NewMockSourceResumeCloser(ctrl)
		src := mock_source.NewMockSourcer(ctrl)
		src.EXPECT().Name().Return("src-1").AnyTimes()

		target.EXPECT().WorkspaceInfo(gomock.Any()).Return(&slack.AuthTestResponse{TeamID: "T01"}, nil)
		src.EXPECT().WorkspaceInfo(gomock.Any()).Return(nil, source.ErrNotFound)
		src.EXPECT().Users(gomock.Any()).Return(nil, source.ErrNotFound)
		src.EXPECT().Channels(gomock.Any()).Return(nil, source.ErrNotFound)

		err := checkArchives(t.Context(), target, src)
		require.NoError(t, err)
	})
}

func TestGetTeamID(t *testing.T) {
	t.Run("uses workspace info when available", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		src := mock_source.NewMockSourcer(ctrl)
		src.EXPECT().WorkspaceInfo(gomock.Any()).Return(&slack.AuthTestResponse{TeamID: "T01"}, nil)

		got, err := getTeamID(t.Context(), src)
		require.NoError(t, err)
		require.Equal(t, "T01", got)
	})

	t.Run("falls back to majority user team", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		src := mock_source.NewMockSourcer(ctrl)
		src.EXPECT().WorkspaceInfo(gomock.Any()).Return(nil, source.ErrNotFound)
		src.EXPECT().Users(gomock.Any()).Return([]slack.User{
			{TeamID: "T01"},
			{TeamID: "T01"},
			{TeamID: "T02"},
		}, nil)

		got, err := getTeamID(t.Context(), src)
		require.NoError(t, err)
		require.Equal(t, "T01", got)
	})

	t.Run("falls back to channel shared team ids", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		src := mock_source.NewMockSourcer(ctrl)
		src.EXPECT().WorkspaceInfo(gomock.Any()).Return(nil, source.ErrNotFound)
		src.EXPECT().Users(gomock.Any()).Return([]slack.User{
			{TeamID: "T01"},
			{TeamID: "T02"},
		}, nil)
		src.EXPECT().Channels(gomock.Any()).Return([]slack.Channel{
			testChannel("C01", "general", "T03"),
		}, nil)

		got, err := getTeamID(t.Context(), src)
		require.NoError(t, err)
		require.Equal(t, "T03", got)
	})

	t.Run("returns workspace info error when not recoverable", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		src := mock_source.NewMockSourcer(ctrl)
		src.EXPECT().WorkspaceInfo(gomock.Any()).Return(nil, errors.New("boom"))

		_, err := getTeamID(t.Context(), src)
		require.Error(t, err)
		require.Contains(t, err.Error(), "workspace info: boom")
	})
}

func TestUsersTeamID(t *testing.T) {
	t.Run("majority team wins", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		src := mock_source.NewMockSourcer(ctrl)
		src.EXPECT().Users(gomock.Any()).Return([]slack.User{
			{TeamID: "T01"},
			{TeamID: "T01"},
			{TeamID: "T02"},
		}, nil)

		got, err := usersTeamID(t.Context(), src)
		require.NoError(t, err)
		require.Equal(t, "T01", got)
	})

	t.Run("no users returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		src := mock_source.NewMockSourcer(ctrl)
		src.EXPECT().Users(gomock.Any()).Return([]slack.User{}, nil)

		_, err := usersTeamID(t.Context(), src)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no users found")
	})

	t.Run("ambiguous team counts returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		src := mock_source.NewMockSourcer(ctrl)
		src.EXPECT().Users(gomock.Any()).Return([]slack.User{
			{TeamID: "T01"},
			{TeamID: "T02"},
		}, nil)

		_, err := usersTeamID(t.Context(), src)
		require.Error(t, err)
		require.Contains(t, err.Error(), "ambiguous team count")
	})
}

func TestChannelsTeamID(t *testing.T) {
	t.Run("returns first eligible shared team id", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		src := mock_source.NewMockSourcer(ctrl)
		src.EXPECT().Channels(gomock.Any()).Return([]slack.Channel{
			testIMChannel("D01", "ignored", "TIM"),
			testChannel("C01", "general", "T01"),
		}, nil)

		got, err := channelsTeamID(t.Context(), src)
		require.NoError(t, err)
		require.Equal(t, "T01", got)
	})

	t.Run("ignores IM, MPIM, and group channels", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		src := mock_source.NewMockSourcer(ctrl)
		src.EXPECT().Channels(gomock.Any()).Return([]slack.Channel{
			testIMChannel("D01", "im", "T01"),
			testMPIMChannel("G01", "mpim", "T02"),
			testGroupChannel("P01", "private", "T03"),
		}, nil)

		_, err := channelsTeamID(t.Context(), src)
		require.ErrorIs(t, err, source.ErrNotFound)
	})

	t.Run("returns not found when no eligible channel exists", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		src := mock_source.NewMockSourcer(ctrl)
		src.EXPECT().Channels(gomock.Any()).Return([]slack.Channel{
			{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "C01"}}},
		}, nil)

		_, err := channelsTeamID(t.Context(), src)
		require.ErrorIs(t, err, source.ErrNotFound)
	})
}

func TestOpen(t *testing.T) {
	t.Run("opens target and all sources", func(t *testing.T) {
		targetDir := newArchiveDir(t)
		writeMergeArchive(t, targetDir, []*chunk.Chunk{testWorkspaceChunk("T01")}, nil)

		sourceDir := newArchiveDir(t)
		writeMergeArchive(t, sourceDir, []*chunk.Chunk{testWorkspaceChunk("T01")}, nil)

		conn, srcs, err := open(t.Context(), targetDir, []string{sourceDir})
		require.NoError(t, err)
		require.Len(t, srcs, 1)
		t.Cleanup(func() {
			require.NoError(t, conn.Close())
			closeSources(srcs)
		})
	})

	t.Run("supports direct sqlite target path", func(t *testing.T) {
		targetDir := newArchiveDir(t)
		targetDB := filepath.Join(targetDir, source.DefaultDBFile)
		writeMergeArchive(t, targetDir, []*chunk.Chunk{testWorkspaceChunk("T01")}, nil)

		sourceDir := newArchiveDir(t)
		writeMergeArchive(t, sourceDir, []*chunk.Chunk{testWorkspaceChunk("T01")}, nil)

		conn, srcs, err := open(t.Context(), targetDB, []string{sourceDir})
		require.NoError(t, err)
		require.Len(t, srcs, 1)
		t.Cleanup(func() {
			require.NoError(t, conn.Close())
			closeSources(srcs)
		})
	})

	t.Run("source open failure returns contextual error", func(t *testing.T) {
		targetDir := newArchiveDir(t)
		writeMergeArchive(t, targetDir, []*chunk.Chunk{testWorkspaceChunk("T01")}, nil)

		missing := filepath.Join(t.TempDir(), "missing-source.sqlite")
		_, _, err := open(t.Context(), targetDir, []string{missing})
		require.Error(t, err)
		require.Contains(t, err.Error(), `opening "`+missing+`"`)
	})
}

func TestMergeSource(t *testing.T) {
	t.Run("encodes messages and copies files for direct sqlite target", func(t *testing.T) {
		ctx := context.Background()
		targetDir := newArchiveDir(t)
		targetDB := filepath.Join(targetDir, source.DefaultDBFile)
		target, err := resolveMergeTarget(targetDB)
		require.NoError(t, err)

		conn, err := ensureDb(ctx, targetDB)
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, conn.Close())
		})

		sourceDir := newArchiveDir(t)
		writeMergeArchive(t, sourceDir, mergeSourceChunks(), mergeSourceFiles())

		src, err := source.Load(ctx, sourceDir)
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, src.Close())
		})

		err = mergeSource(ctx, target, conn, src, true, false)
		require.NoError(t, err)

		assertMessageAndThreadCopied(t, targetDir)
		assertFileContents(t, filepath.Join(targetDir, testTopFilePath()), "top attachment")
		assertFileContents(t, filepath.Join(targetDir, testReplyFilePath()), "thread attachment")
	})

	t.Run("does not copy files when files disabled", func(t *testing.T) {
		ctx := context.Background()
		targetDir := newArchiveDir(t)
		target, err := resolveMergeTarget(targetDir)
		require.NoError(t, err)

		conn, err := ensureDb(ctx, targetDir)
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, conn.Close())
		})

		sourceDir := newArchiveDir(t)
		writeMergeArchive(t, sourceDir, mergeSourceChunks(), mergeSourceFiles())

		src, err := source.Load(ctx, sourceDir)
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, src.Close())
		})

		err = mergeSource(ctx, target, conn, src, false, false)
		require.NoError(t, err)

		assertMessageAndThreadCopied(t, targetDir)
		_, err = os.Stat(filepath.Join(targetDir, testTopFilePath()))
		require.ErrorIs(t, err, os.ErrNotExist)
		_, err = os.Stat(filepath.Join(targetDir, testReplyFilePath()))
		require.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("copies avatars when avatars enabled", func(t *testing.T) {
		ctx := context.Background()
		targetDir := newArchiveDir(t)
		target, err := resolveMergeTarget(targetDir)
		require.NoError(t, err)

		conn, err := ensureDb(ctx, targetDir)
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, conn.Close())
		})

		sourceDir := newArchiveDir(t)
		writeMergeArchive(t, sourceDir, mergeAvatarChunks(), mergeAvatarFiles())

		src, err := source.Load(ctx, sourceDir)
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, src.Close())
		})

		err = mergeSource(ctx, target, conn, src, false, true)
		require.NoError(t, err)

		assertFileContents(t, filepath.Join(targetDir, testAvatarPath()), "avatar bytes")
	})
}

func TestRunMerge(t *testing.T) {
	t.Run("requires target and at least one source", func(t *testing.T) {
		restoreMergeGlobals(t)

		err := runMerge(t.Context(), cmdMerge, []string{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "expected target database and at least one source")
	})

	t.Run("check only validates and does not mutate target", func(t *testing.T) {
		restoreMergeGlobals(t)
		checkOnly = true

		targetDir := newArchiveDir(t)
		writeMergeArchive(t, targetDir, []*chunk.Chunk{testWorkspaceChunk("T01")}, nil)

		sourceDir := newArchiveDir(t)
		writeMergeArchive(t, sourceDir, mergeSourceChunks(), mergeSourceFiles())

		err := runMerge(t.Context(), cmdMerge, []string{targetDir, sourceDir})
		require.NoError(t, err)

		targetSrc := mustLoadSource(t, targetDir)
		t.Cleanup(func() {
			require.NoError(t, targetSrc.Close())
		})

		_, err = targetSrc.Channels(t.Context())
		require.ErrorIs(t, err, source.ErrNotFound)
	})

	t.Run("successful end to end merge with direct sqlite target", func(t *testing.T) {
		restoreMergeGlobals(t)
		checkOnly = false

		targetDir := newArchiveDir(t)
		targetDB := filepath.Join(targetDir, source.DefaultDBFile)
		writeMergeArchive(t, targetDir, []*chunk.Chunk{testWorkspaceChunk("T01")}, nil)

		sourceDir := newArchiveDir(t)
		writeMergeArchive(t, sourceDir, mergeSourceChunks(), mergeSourceFiles())

		err := runMerge(t.Context(), cmdMerge, []string{targetDB, sourceDir})
		require.NoError(t, err)

		assertMessageAndThreadCopied(t, targetDir)
		assertFileContents(t, filepath.Join(targetDir, testReplyFilePath()), "thread attachment")
	})
}

const (
	testTeamID    = "T01"
	testChannelID = "C01"
)

func restoreMergeGlobals(t *testing.T) {
	t.Helper()
	prevCheckOnly := checkOnly
	prevFlags := mergeFlags
	t.Cleanup(func() {
		checkOnly = prevCheckOnly
		mergeFlags = prevFlags
	})
}

func writeMergeArchive(t *testing.T, archiveDir string, chunks []*chunk.Chunk, files map[string]string) {
	t.Helper()

	conn, err := bootstrap.Database(archiveDir)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})

	dbp, err := dbase.New(t.Context(), conn, bootstrap.SessionInfo("test"))
	require.NoError(t, err)
	for _, ch := range chunks {
		require.NoError(t, dbp.Encode(t.Context(), ch))
	}
	require.NoError(t, dbp.Finish())

	for rel, content := range files {
		full := filepath.Join(archiveDir, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}
}

func mustLoadSource(t *testing.T, path string) source.SourceResumeCloser {
	t.Helper()
	src, err := source.Load(t.Context(), path)
	require.NoError(t, err)
	return src
}

func testWorkspaceChunk(teamID string) *chunk.Chunk {
	return &chunk.Chunk{
		Type: chunk.CWorkspaceInfo,
		WorkspaceInfo: &slack.AuthTestResponse{
			URL:    "https://example.slack.com/",
			Team:   "Example",
			TeamID: teamID,
			User:   "tester",
			UserID: "U01",
		},
	}
}

func testUsersChunk(users ...slack.User) *chunk.Chunk {
	return &chunk.Chunk{
		Type:   chunk.CUsers,
		Users:  users,
		IsLast: true,
		Count:  int32(len(users)),
	}
}

func testChannelsChunk(channels ...slack.Channel) *chunk.Chunk {
	return &chunk.Chunk{
		Type:     chunk.CChannels,
		Channels: channels,
		IsLast:   true,
		Count:    int32(len(channels)),
	}
}

func testMessagesChunk(channelID string, msgs ...slack.Message) *chunk.Chunk {
	return &chunk.Chunk{
		Type:      chunk.CMessages,
		ChannelID: channelID,
		Messages:  msgs,
		IsLast:    true,
		Count:     int32(len(msgs)),
	}
}

func testThreadChunk(channelID string, parent slack.Message, msgs ...slack.Message) *chunk.Chunk {
	return &chunk.Chunk{
		Type:      chunk.CThreadMessages,
		ChannelID: channelID,
		ThreadTS:  parent.Timestamp,
		Parent:    &parent,
		Messages:  msgs,
		IsLast:    true,
		Count:     int32(len(msgs)),
	}
}

func mergeSourceChunks() []*chunk.Chunk {
	root := testRootMessage()
	top := testTopLevelFileMessage()
	reply := testReplyMessage()
	return []*chunk.Chunk{
		testWorkspaceChunk(testTeamID),
		testChannelsChunk(testChannel(testChannelID, "general", testTeamID)),
		testMessagesChunk(testChannelID, root, top),
		testThreadChunk(testChannelID, root, reply),
	}
}

func mergeSourceFiles() map[string]string {
	return map[string]string{
		testTopFilePath():   "top attachment",
		testReplyFilePath(): "thread attachment",
	}
}

func mergeAvatarChunks() []*chunk.Chunk {
	return []*chunk.Chunk{
		testWorkspaceChunk(testTeamID),
		testUsersChunk(testAvatarUser()),
		testChannelsChunk(testChannel(testChannelID, "general", testTeamID)),
	}
}

func mergeAvatarFiles() map[string]string {
	return map[string]string{
		testAvatarPath(): "avatar bytes",
	}
}

func testChannel(id, name, teamID string) slack.Channel {
	return slack.Channel{
		GroupConversation: slack.GroupConversation{
			Conversation: slack.Conversation{
				ID:            id,
				SharedTeamIDs: []string{teamID},
			},
			Name: name,
		},
	}
}

func testIMChannel(id, name, teamID string) slack.Channel {
	ch := testChannel(id, name, teamID)
	ch.IsIM = true
	return ch
}

func testMPIMChannel(id, name, teamID string) slack.Channel {
	ch := testChannel(id, name, teamID)
	ch.IsMpIM = true
	return ch
}

func testGroupChannel(id, name, teamID string) slack.Channel {
	ch := testChannel(id, name, teamID)
	ch.IsGroup = true
	return ch
}

func testRootMessage() slack.Message {
	return slack.Message{
		Msg: slack.Msg{
			Timestamp:       "1710000000.000001",
			ThreadTimestamp: "1710000000.000001",
			ReplyCount:      1,
			Text:            "root",
		},
	}
}

func testTopLevelFileMessage() slack.Message {
	return slack.Message{
		Msg: slack.Msg{
			Timestamp: "1710000002.000001",
			Text:      "top file",
			Files: []slack.File{{
				ID:                 "F-top",
				Name:               "top.txt",
				URLPrivateDownload: "unused",
			}},
		},
	}
}

func testReplyMessage() slack.Message {
	return slack.Message{
		Msg: slack.Msg{
			Timestamp:       "1710000001.000001",
			ThreadTimestamp: "1710000000.000001",
			Text:            "reply",
			Files: []slack.File{{
				ID:                 "F-thread",
				Name:               "reply.txt",
				URLPrivateDownload: "unused",
			}},
		},
	}
}

func testAvatarUser() slack.User {
	return slack.User{
		ID: "U-avatar",
		Profile: slack.UserProfile{
			ImageOriginal: "https://example.com/avatar.jpg",
		},
	}
}

func testTopFilePath() string {
	return filepath.Join(chunk.UploadsDir, "F-top", "top.txt")
}

func testReplyFilePath() string {
	return filepath.Join(chunk.UploadsDir, "F-thread", "reply.txt")
}

func testAvatarPath() string {
	uid, file := source.AvatarParams(&[]slack.User{testAvatarUser()}[0])
	return filepath.Join(chunk.AvatarsDir, uid, file)
}

func assertMessageAndThreadCopied(t *testing.T, archiveDir string) {
	t.Helper()

	merged := mustLoadSource(t, archiveDir)
	t.Cleanup(func() {
		require.NoError(t, merged.Close())
	})

	msgIter, err := merged.AllMessages(t.Context(), testChannelID)
	require.NoError(t, err)
	var messages []slack.Message
	for msg, err := range msgIter {
		require.NoError(t, err)
		messages = append(messages, msg)
	}
	require.Len(t, messages, 2)

	threadIter, err := merged.AllThreadMessages(t.Context(), testChannelID, testRootMessage().Timestamp)
	require.NoError(t, err)
	var threadMsgs []slack.Message
	for msg, err := range threadIter {
		require.NoError(t, err)
		threadMsgs = append(threadMsgs, msg)
	}
	require.Len(t, threadMsgs, 2)
	require.Len(t, threadMsgs[1].Files, 1)
	require.Equal(t, "F-thread", threadMsgs[1].Files[0].ID)
}

func assertFileContents(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, want, string(data))
}
