package diag

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/require"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v4/internal/chunk"
	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase"
	"github.com/rusq/slackdump/v4/source"
)

func TestMergeSource_CopiesThreadReplyFilesForDirectSQLiteTarget(t *testing.T) {
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
	writeMergeSourceFixture(t, sourceDir)

	src, err := source.Load(ctx, sourceDir)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, src.Close())
	})

	err = mergeSource(ctx, target, conn, src, true, false)
	require.NoError(t, err)

	replyPath := filepath.Join(targetDir, chunk.UploadsDir, "F-thread", "reply.txt")
	data, err := os.ReadFile(replyPath)
	require.NoError(t, err)
	require.Equal(t, "thread attachment", string(data))

	merged, err := source.Load(ctx, targetDir)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, merged.Close())
	})

	threadIter, err := merged.AllThreadMessages(ctx, "C01", "1710000000.000001")
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

func writeMergeSourceFixture(t *testing.T, archiveDir string) {
	t.Helper()

	conn, err := bootstrap.Database(archiveDir)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})

	dbp, err := dbase.New(t.Context(), conn, bootstrap.SessionInfo("test"))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, dbp.Close())
	})

	channel := slack.Channel{
		GroupConversation: slack.GroupConversation{
			Conversation: slack.Conversation{
				ID:            "C01",
				SharedTeamIDs: []string{"T01"},
			},
			Name: "general",
		},
	}
	root := slack.Message{
		Msg: slack.Msg{
			Timestamp:       "1710000000.000001",
			ThreadTimestamp: "1710000000.000001",
			ReplyCount:      1,
			Text:            "root",
		},
	}
	reply := slack.Message{
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

	require.NoError(t, dbp.Encode(t.Context(), &chunk.Chunk{
		Type:     chunk.CChannels,
		Channels: []slack.Channel{channel},
		IsLast:   true,
		Count:    1,
	}))
	require.NoError(t, dbp.Encode(t.Context(), &chunk.Chunk{
		Type:      chunk.CMessages,
		ChannelID: channel.ID,
		Messages:  []slack.Message{root},
		IsLast:    true,
		Count:     1,
	}))
	require.NoError(t, dbp.Encode(t.Context(), &chunk.Chunk{
		Type:      chunk.CThreadMessages,
		ChannelID: channel.ID,
		ThreadTS:  root.Timestamp,
		Parent:    &root,
		Messages:  []slack.Message{reply},
		IsLast:    true,
		Count:     1,
	}))

	uploadDir := filepath.Join(archiveDir, chunk.UploadsDir, "F-thread")
	require.NoError(t, os.MkdirAll(uploadDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(uploadDir, "reply.txt"), []byte("thread attachment"), 0o644))
}
