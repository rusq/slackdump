// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package dbase

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/rusq/slack"
	"github.com/stretchr/testify/require"

	"github.com/rusq/slackdump/v4/internal/chunk"
	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase/repository"
)

func TestPostgresArchiveResume(t *testing.T) {
	databaseURL := os.Getenv("SLACKDUMP_POSTGRES_TEST_URL")
	if databaseURL == "" {
		t.Skip("SLACKDUMP_POSTGRES_TEST_URL is not set")
	}

	ctx := t.Context()
	admin, err := sqlx.Open(repository.PostgresDriver, databaseURL)
	require.NoError(t, err)
	defer admin.Close()
	require.NoError(t, admin.PingContext(ctx))

	schema := fmt.Sprintf("slackdump_test_%d", time.Now().UnixNano())
	_, err = admin.ExecContext(ctx, "CREATE SCHEMA "+pq.QuoteIdentifier(schema))
	require.NoError(t, err)
	defer func() {
		_, _ = admin.ExecContext(ctx, "DROP SCHEMA "+pq.QuoteIdentifier(schema)+" CASCADE")
	}()

	conn, err := sqlx.Open(repository.PostgresDriver, postgresSearchPath(databaseURL, schema))
	require.NoError(t, err)
	defer conn.Close()

	writer, err := New(ctx, conn, SessionInfo{Mode: "archive"})
	require.NoError(t, err)
	require.EqualValues(t, 1, writer.sessionID)

	workspace := &slack.AuthTestResponse{
		Team:   "Test Workspace",
		TeamID: "T123",
		UserID: "U123",
		URL:    "https://example.slack.com/",
	}
	_, err = writer.InsertChunk(ctx, &chunk.Chunk{
		Type:          chunk.CWorkspaceInfo,
		Timestamp:     time.Now().Unix(),
		Count:         1,
		IsLast:        true,
		WorkspaceInfo: workspace,
	})
	require.NoError(t, err)

	messageTS := "1700000000.000001"
	_, err = writer.InsertChunk(ctx, &chunk.Chunk{
		Type:      chunk.CMessages,
		Timestamp: time.Now().Unix(),
		ChannelID: "C123",
		Count:     1,
		IsLast:    true,
		Messages: []slack.Message{{
			Msg: slack.Msg{Timestamp: messageTS, Text: "postgres integration"},
		}},
	})
	require.NoError(t, err)

	threadTS := "1700000001.000001"
	replyTS := "1700000002.000001"
	threadParent := slack.Message{Msg: slack.Msg{
		Timestamp:       threadTS,
		ThreadTimestamp: threadTS,
		LatestReply:     replyTS,
		Text:            "thread parent",
	}}
	_, err = writer.InsertChunk(ctx, &chunk.Chunk{
		Type:       chunk.CMessages,
		Timestamp:  time.Now().Unix(),
		ChannelID:  "CTHREAD",
		Count:      1,
		IsLast:     true,
		Messages:   []slack.Message{threadParent},
		ThreadOnly: false,
	})
	require.NoError(t, err)
	_, err = writer.InsertChunk(ctx, &chunk.Chunk{
		Type:      chunk.CThreadMessages,
		Timestamp: time.Now().Unix(),
		ChannelID: "CTHREAD",
		Count:     2,
		IsLast:    true,
		Messages: []slack.Message{
			threadParent,
			{Msg: slack.Msg{Timestamp: replyTS, ThreadTimestamp: threadTS, SubType: "thread_broadcast", Text: "thread reply"}},
		},
		ThreadOnly: false,
	})
	require.NoError(t, err)
	require.NoError(t, writer.Finish())

	src := SourceFromConnection(conn, false)
	gotWorkspace, err := src.WorkspaceInfo(ctx)
	require.NoError(t, err)
	require.Equal(t, workspace.TeamID, gotWorkspace.TeamID)
	latest, err := src.Latest(ctx)
	require.NoError(t, err)
	require.Len(t, latest, 3)
	threadCount, err := repository.NewMessageRepository().CountThread(ctx, conn, "CTHREAD", threadTS)
	require.NoError(t, err)
	require.EqualValues(t, 2, threadCount)

	resumeWriter, err := New(ctx, conn, SessionInfo{Mode: "resume"})
	require.NoError(t, err)
	require.EqualValues(t, 2, resumeWriter.sessionID)
	require.NoError(t, resumeWriter.Finish())
}

func postgresSearchPath(databaseURL, schema string) string {
	if parsed, err := url.Parse(databaseURL); err == nil && parsed.Scheme != "" {
		query := parsed.Query()
		query.Set("search_path", schema)
		parsed.RawQuery = query.Encode()
		return parsed.String()
	}
	if strings.TrimSpace(databaseURL) == "" {
		return "search_path=" + schema
	}
	return databaseURL + " search_path=" + schema
}
