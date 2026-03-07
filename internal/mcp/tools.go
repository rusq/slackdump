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

package mcp

// In this file: MCP tool definitions and handler implementations.

import (
	"context"
	"errors"
	"fmt"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	mcpsrv "github.com/mark3labs/mcp-go/server"

	"github.com/rusq/slackdump/v4/source"
)

// errNoSource is returned by tool handlers when no source has been loaded yet.
var errNoSource = errors.New("no archive is loaded; call load_source first")

// ─── load_source ──────────────────────────────────────────────────────────────

func (s *Server) toolLoadSource() mcpsrv.ServerTool {
	tool := mcplib.NewTool("load_source",
		mcplib.WithDescription(`Open a Slackdump archive as the active data source.

Calling this tool closes the currently open archive (if any) and opens the
archive at the given path.  Only one source may be open at any time.  After
a successful call all data tools (list_channels, list_users, get_messages,
get_thread, get_workspace_info) operate on the newly opened archive.

Accepted archive formats: SQLite database (.db/.sqlite), Slackdump chunk
directory, Slackdump dump directory or ZIP, Slack export directory or ZIP.`),
		mcplib.WithString("path",
			mcplib.Description("Filesystem path to the Slackdump archive file or directory to open."),
			mcplib.Required(),
		),
	)
	return mcpsrv.ServerTool{Tool: tool, Handler: s.handleLoadSource}
}

func (s *Server) handleLoadSource(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	path, ok := stringArg(req, "path")
	if !ok || path == "" {
		return resultErr(errors.New("load_source: path is required")), nil
	}

	s.logger.InfoContext(ctx, "mcp: load_source: opening archive", "path", path)

	next, err := s.loader(ctx, path)
	if err != nil {
		return resultErr(fmt.Errorf("load_source: open %q: %w", path, err)), nil
	}

	if err := s.loadSource(next); err != nil {
		// loadSource itself never returns an error currently, but be defensive.
		_ = next.Close()
		return resultErr(fmt.Errorf("load_source: %w", err)), nil
	}

	s.logger.InfoContext(ctx, "mcp: load_source: archive opened", "path", path, "type", next.Type())
	return resultText(fmt.Sprintf(
		"Archive %q (type: %s) loaded successfully. You can now use the data tools.",
		next.Name(), next.Type(),
	)), nil
}

// ─── list_channels ────────────────────────────────────────────────────────────

func (s *Server) toolListChannels() mcpsrv.ServerTool {
	tool := mcplib.NewTool("list_channels",
		mcplib.WithDescription("List all channels (conversations) present in the Slackdump archive. Returns channel IDs, names, types, and member counts."),
		mcplib.WithReadOnlyHintAnnotation(true),
	)
	return mcpsrv.ServerTool{Tool: tool, Handler: s.handleListChannels}
}

// channelSummary is a JSON-serialisable summary of a Slack channel.
type channelSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	IsChannel   bool   `json:"is_channel,omitempty"`
	IsGroup     bool   `json:"is_group,omitempty"`
	IsIM        bool   `json:"is_im,omitempty"`
	IsMPIM      bool   `json:"is_mpim,omitempty"`
	IsArchived  bool   `json:"is_archived,omitempty"`
	MemberCount int    `json:"member_count,omitempty"`
	Topic       string `json:"topic,omitempty"`
	Purpose     string `json:"purpose,omitempty"`
}

func (s *Server) handleListChannels(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	src := s.source()
	if src == nil {
		return resultErr(errNoSource), nil
	}

	channels, err := src.Channels(ctx)
	if err != nil {
		if errors.Is(err, source.ErrNotSupported) {
			return resultText("This archive type does not support listing channels."), nil
		}
		return resultErr(fmt.Errorf("list_channels: %w", err)), nil
	}

	summaries := make([]channelSummary, 0, len(channels))
	for _, c := range channels {
		topic := ""
		if c.Topic.Value != "" {
			topic = c.Topic.Value
		}
		purpose := ""
		if c.Purpose.Value != "" {
			purpose = c.Purpose.Value
		}
		summaries = append(summaries, channelSummary{
			ID:          c.ID,
			Name:        c.Name,
			IsChannel:   c.IsChannel,
			IsGroup:     c.IsGroup,
			IsIM:        c.IsIM,
			IsMPIM:      c.IsMpIM,
			IsArchived:  c.IsArchived,
			MemberCount: c.NumMembers,
			Topic:       topic,
			Purpose:     purpose,
		})
	}

	result, err := resultJSON(summaries)
	if err != nil {
		return resultErr(fmt.Errorf("list_channels: serialise: %w", err)), nil
	}
	return result, nil
}

// ─── get_channel ──────────────────────────────────────────────────────────────

func (s *Server) toolGetChannel() mcpsrv.ServerTool {
	tool := mcplib.NewTool("get_channel",
		mcplib.WithDescription("Get detailed information about a specific channel by its ID."),
		mcplib.WithString("channel_id",
			mcplib.Description("The Slack channel ID (e.g. C01234ABCD)"),
			mcplib.Required(),
		),
		mcplib.WithReadOnlyHintAnnotation(true),
	)
	return mcpsrv.ServerTool{Tool: tool, Handler: s.handleGetChannel}
}

func (s *Server) handleGetChannel(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	src := s.source()
	if src == nil {
		return resultErr(errNoSource), nil
	}

	channelID, ok := stringArg(req, "channel_id")
	if !ok || channelID == "" {
		return resultErr(errors.New("get_channel: channel_id is required")), nil
	}

	ch, err := src.ChannelInfo(ctx, channelID)
	if err != nil {
		if errors.Is(err, source.ErrNotFound) {
			return resultText(fmt.Sprintf("Channel %q not found in archive.", channelID)), nil
		}
		if errors.Is(err, source.ErrNotSupported) {
			return resultText("This archive type does not support channel info lookup."), nil
		}
		return resultErr(fmt.Errorf("get_channel: %w", err)), nil
	}

	result, err := resultJSON(ch)
	if err != nil {
		return resultErr(fmt.Errorf("get_channel: serialise: %w", err)), nil
	}
	return result, nil
}

// ─── list_users ───────────────────────────────────────────────────────────────

func (s *Server) toolListUsers() mcpsrv.ServerTool {
	tool := mcplib.NewTool("list_users",
		mcplib.WithDescription("List all users/members in the Slackdump archive. Returns user IDs, display names, real names, and email addresses."),
		mcplib.WithReadOnlyHintAnnotation(true),
	)
	return mcpsrv.ServerTool{Tool: tool, Handler: s.handleListUsers}
}

// userSummary is a JSON-serialisable summary of a Slack user.
type userSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	RealName    string `json:"real_name,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Email       string `json:"email,omitempty"`
	IsBot       bool   `json:"is_bot,omitempty"`
	IsDeleted   bool   `json:"is_deleted,omitempty"`
	TZ          string `json:"tz,omitempty"`
}

func (s *Server) handleListUsers(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	src := s.source()
	if src == nil {
		return resultErr(errNoSource), nil
	}

	users, err := src.Users(ctx)
	if err != nil {
		if errors.Is(err, source.ErrNotSupported) {
			return resultText("This archive type does not support listing users."), nil
		}
		return resultErr(fmt.Errorf("list_users: %w", err)), nil
	}

	summaries := make([]userSummary, 0, len(users))
	for _, u := range users {
		summaries = append(summaries, userSummary{
			ID:          u.ID,
			Name:        u.Name,
			RealName:    u.RealName,
			DisplayName: u.Profile.DisplayName,
			Email:       u.Profile.Email,
			IsBot:       u.IsBot,
			IsDeleted:   u.Deleted,
			TZ:          u.TZ,
		})
	}

	result, err := resultJSON(summaries)
	if err != nil {
		return resultErr(fmt.Errorf("list_users: serialise: %w", err)), nil
	}
	return result, nil
}

// ─── get_messages ─────────────────────────────────────────────────────────────

func (s *Server) toolGetMessages() mcpsrv.ServerTool {
	tool := mcplib.NewTool("get_messages",
		mcplib.WithDescription(`Retrieve messages from a channel in the Slackdump archive.

Returns messages sorted by timestamp in ascending order. To page through messages use
the 'after_ts' parameter (set it to the Timestamp of the last message received).
Thread reply counts are included but thread bodies are not; use get_thread for those.`),
		mcplib.WithString("channel_id",
			mcplib.Description("The Slack channel ID to read messages from (e.g. C01234ABCD)"),
			mcplib.Required(),
		),
		mcplib.WithNumber("limit",
			mcplib.Description("Maximum number of messages to return (1–1000, default 100)"),
		),
		mcplib.WithString("after_ts",
			mcplib.Description("Return only messages with a timestamp strictly after this value (Slack ts format, e.g. \"1609459200.000001\"). Use for pagination."),
		),
		mcplib.WithReadOnlyHintAnnotation(true),
	)
	return mcpsrv.ServerTool{Tool: tool, Handler: s.handleGetMessages}
}

// messageSummary is a JSON-serialisable summary of a Slack message.
type messageSummary struct {
	Timestamp  string `json:"ts"`
	UserID     string `json:"user,omitempty"`
	Text       string `json:"text,omitempty"`
	ReplyCount int    `json:"reply_count,omitempty"`
	ThreadTS   string `json:"thread_ts,omitempty"`
	Subtype    string `json:"subtype,omitempty"`
}

const (
	defLimit = 100
	minLimit = 1
	maxLimit = 1000
)

func (s *Server) handleGetMessages(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	src := s.source()
	if src == nil {
		return resultErr(errNoSource), nil
	}

	channelID, ok := stringArg(req, "channel_id")
	if !ok || channelID == "" {
		return resultErr(errors.New("get_messages: channel_id is required")), nil
	}

	limit := intArg(req, "limit", defLimit)
	limit = max(min(limit, maxLimit), minLimit) // ensure within bounds

	afterTS, _ := stringArg(req, "after_ts")

	iter, err := src.AllMessages(ctx, channelID)
	if err != nil {
		if errors.Is(err, source.ErrNotFound) {
			return resultText(fmt.Sprintf("No messages found for channel %q.", channelID)), nil
		}
		if errors.Is(err, source.ErrNotSupported) {
			return resultText("This archive type does not support reading messages."), nil
		}
		return resultErr(fmt.Errorf("get_messages: %w", err)), nil
	}

	var msgs []messageSummary
	for msg, err := range iter {
		if err != nil {
			return resultErr(fmt.Errorf("get_messages: iterate: %w", err)), nil
		}
		// Skip thread replies (they have a ThreadTimestamp different from Timestamp).
		if msg.ThreadTimestamp != "" && msg.ThreadTimestamp != msg.Timestamp {
			continue
		}
		// Apply after_ts filter.
		if afterTS != "" && msg.Timestamp <= afterTS {
			continue
		}
		msgs = append(msgs, messageSummary{
			Timestamp:  msg.Timestamp,
			UserID:     msg.User,
			Text:       msg.Text,
			ReplyCount: msg.ReplyCount,
			ThreadTS:   msg.ThreadTimestamp,
			Subtype:    msg.SubType,
		})
		if len(msgs) >= limit {
			break
		}
	}

	result, err := resultJSON(msgs)
	if err != nil {
		return resultErr(fmt.Errorf("get_messages: serialise: %w", err)), nil
	}
	return result, nil
}

// ─── get_thread ───────────────────────────────────────────────────────────────

func (s *Server) toolGetThread() mcpsrv.ServerTool {
	tool := mcplib.NewTool("get_thread",
		mcplib.WithDescription("Retrieve all messages in a thread (including the parent message) from a Slackdump archive."),
		mcplib.WithString("channel_id",
			mcplib.Description("The Slack channel ID that contains the thread (e.g. C01234ABCD)"),
			mcplib.Required(),
		),
		mcplib.WithString("thread_ts",
			mcplib.Description("The timestamp of the parent message / thread ID (Slack ts format, e.g. \"1609459200.000001\")"),
			mcplib.Required(),
		),
		mcplib.WithReadOnlyHintAnnotation(true),
	)
	return mcpsrv.ServerTool{Tool: tool, Handler: s.handleGetThread}
}

func (s *Server) handleGetThread(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	src := s.source()
	if src == nil {
		return resultErr(errNoSource), nil
	}

	channelID, ok := stringArg(req, "channel_id")
	if !ok || channelID == "" {
		return resultErr(errors.New("get_thread: channel_id is required")), nil
	}
	threadTS, ok := stringArg(req, "thread_ts")
	if !ok || threadTS == "" {
		return resultErr(errors.New("get_thread: thread_ts is required")), nil
	}

	iter, err := src.AllThreadMessages(ctx, channelID, threadTS)
	if err != nil {
		if errors.Is(err, source.ErrNotFound) {
			return resultText(fmt.Sprintf("Thread %q in channel %q not found.", threadTS, channelID)), nil
		}
		if errors.Is(err, source.ErrNotSupported) {
			return resultText("This archive type does not support reading threads."), nil
		}
		return resultErr(fmt.Errorf("get_thread: %w", err)), nil
	}

	var msgs []messageSummary
	for msg, err := range iter {
		if err != nil {
			return resultErr(fmt.Errorf("get_thread: iterate: %w", err)), nil
		}
		msgs = append(msgs, messageSummary{
			Timestamp:  msg.Timestamp,
			UserID:     msg.User,
			Text:       msg.Text,
			ReplyCount: msg.ReplyCount,
			ThreadTS:   msg.ThreadTimestamp,
			Subtype:    msg.SubType,
		})
	}

	result, err := resultJSON(msgs)
	if err != nil {
		return resultErr(fmt.Errorf("get_thread: serialise: %w", err)), nil
	}
	return result, nil
}

// ─── get_workspace_info ───────────────────────────────────────────────────────

func (s *Server) toolGetWorkspaceInfo() mcpsrv.ServerTool {
	tool := mcplib.NewTool("get_workspace_info",
		mcplib.WithDescription("Return workspace/team information stored in the archive, such as team name, domain, and the authenticated user."),
		mcplib.WithReadOnlyHintAnnotation(true),
	)
	return mcpsrv.ServerTool{Tool: tool, Handler: s.handleGetWorkspaceInfo}
}

func (s *Server) handleGetWorkspaceInfo(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	src := s.source()
	if src == nil {
		return resultErr(errNoSource), nil
	}

	info, err := src.WorkspaceInfo(ctx)
	if err != nil {
		if errors.Is(err, source.ErrNotFound) || errors.Is(err, source.ErrNotSupported) {
			return resultText("Workspace information is not available in this archive."), nil
		}
		return resultErr(fmt.Errorf("get_workspace_info: %w", err)), nil
	}

	result, err := resultJSON(info)
	if err != nil {
		return resultErr(fmt.Errorf("get_workspace_info: serialise: %w", err)), nil
	}
	return result, nil
}
