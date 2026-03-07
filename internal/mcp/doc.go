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

// Package mcp implements a Model Context Protocol (MCP) server for slackdump.
// It exposes Slackdump archive data through MCP tools that AI agents can call
// to inspect, search and summarise Slack conversations stored in any slackdump
// archive format (SQLite database, chunk directory, export ZIP, dump JSON).
//
// The server is intentionally read-only: it never writes to or modifies the
// underlying archive.
//
// Transport: the server supports two transports selectable at runtime:
//   - stdio  – standard MCP stdio transport (default); suitable for local
//     agent integration (e.g. Claude Desktop, VS Code Copilot).
//   - http   – Streamable HTTP transport; suitable for remote agents or when
//     multiple concurrent clients are needed.
package mcp
