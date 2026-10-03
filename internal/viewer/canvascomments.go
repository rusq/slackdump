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

package viewer

import (
	"context"
	"fmt"
	"iter"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/internal/structures"
)

// canvasCommentSource is an optional source capability. Slack export sources
// intentionally do not implement it.
type canvasCommentSource interface {
	CanvasMessages(ctx context.Context, hiddenChannelID string) (iter.Seq2[slack.Message, error], error)
	CanvasThreadMessages(ctx context.Context, hiddenChannelID, threadTS string) (iter.Seq2[slack.Message, error], error)
}

func canvasHiddenChannelID(ci *slack.Channel) (string, error) {
	fileID, ok := structures.CanvasFileID(ci)
	if !ok {
		return "", fmt.Errorf("channel %s canvas file ID: invalid or missing", ci.ID)
	}
	hiddenChannelID, ok := structures.CanvasChannelID(fileID)
	if !ok {
		return "", fmt.Errorf("channel %s canvas file ID %q: invalid", ci.ID, fileID)
	}
	return hiddenChannelID, nil
}

func collectMessages(it iter.Seq2[slack.Message, error]) ([]slack.Message, error) {
	if it == nil {
		return nil, nil
	}
	var messages []slack.Message
	for msg, err := range it {
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

func (v *Viewer) canvasCommentRoots(ctx context.Context, ci *slack.Channel) ([]slack.Message, bool, error) {
	src, supported := v.src.(canvasCommentSource)
	if !supported {
		return nil, false, nil
	}
	if ci == nil || ci.Properties == nil || ci.Properties.Canvas.IsEmpty || ci.Properties.Canvas.FileId == "" {
		return nil, true, nil
	}
	hiddenChannelID, err := canvasHiddenChannelID(ci)
	if err != nil {
		return nil, true, err
	}
	it, err := src.CanvasMessages(ctx, hiddenChannelID)
	if err != nil {
		return nil, true, err
	}
	roots, err := collectMessages(it)
	return roots, true, err
}

func (v *Viewer) canvasCommentThread(ctx context.Context, ci *slack.Channel, threadTS string) ([]slack.Message, bool, error) {
	src, supported := v.src.(canvasCommentSource)
	if !supported {
		return nil, false, nil
	}
	if ci == nil || ci.Properties == nil || ci.Properties.Canvas.IsEmpty || ci.Properties.Canvas.FileId == "" {
		return nil, true, nil
	}
	hiddenChannelID, err := canvasHiddenChannelID(ci)
	if err != nil {
		return nil, true, err
	}
	it, err := src.CanvasThreadMessages(ctx, hiddenChannelID, threadTS)
	if err != nil {
		return nil, true, err
	}
	messages, err := collectMessages(it)
	return messages, true, err
}
