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
