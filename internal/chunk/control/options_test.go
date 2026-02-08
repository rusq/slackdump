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
package control

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v3/internal/chunk/control/mock_control"
	"github.com/rusq/slackdump/v3/mocks/mock_processor"
)

func TestOptions(t *testing.T) {
	var (
		filer       = &mock_processor.MockFiler{}
		avatar      = &mock_processor.MockAvatars{}
		logger      = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))
		transformer = &mock_control.MockExportTransformer{}
		flags       = Flags{
			MemberOnly:   true,
			RecordFiles:  true,
			Refresh:      true,
			ChannelUsers: true,
			ChannelTypes: []string{"public_channel"},
		}
	)
	var (
		optFiler = WithFiler(filer)
		optAv    = WithAvatarProcessor(avatar)
		optLg    = WithLogger(logger)
		optTf    = WithCoordinator(transformer)
		optFl    = WithFlags(flags)
	)
	t.Run("WithFiler", func(t *testing.T) {
		o := &options{}
		optFiler(o)
		assert.Equal(t, filer, o.filer)
	})
	t.Run("WithAvatars", func(t *testing.T) {
		o := &options{}
		optAv(o)
		assert.Equal(t, avatar, o.avp)
	})
	t.Run("WithLogger", func(t *testing.T) {
		o := &options{}
		optLg(o)
		assert.Equal(t, logger, o.lg)
	})
	t.Run("WithTransformer", func(t *testing.T) {
		o := &options{}
		optTf(o)
		assert.Equal(t, transformer, o.tf)
	})
	t.Run("WithFlags", func(t *testing.T) {
		o := &options{}
		optFl(o)
		assert.Equal(t, flags, o.flags)
	})
}
