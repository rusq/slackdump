package dedupe

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase/repository"
)

type stubRepo struct {
	previewFn func(context.Context, *sqlx.DB) (repository.DedupeStats, error)
	dedupeFn  func(context.Context, *sqlx.DB) (repository.DedupeStats, error)
}

func (s stubRepo) Preview(ctx context.Context, db *sqlx.DB) (repository.DedupeStats, error) {
	return s.previewFn(ctx, db)
}

func (s stubRepo) Deduplicate(ctx context.Context, db *sqlx.DB) (repository.DedupeStats, error) {
	return s.dedupeFn(ctx, db)
}

func TestRun(t *testing.T) {
	oldNewRepo := newRepo
	t.Cleanup(func() { newRepo = oldNewRepo })

	t.Run("preview only with report", func(t *testing.T) {
		var called bool
		newRepo = func(repository.MessageDedupeMode) repository.DedupeRepository {
			return stubRepo{
				previewFn: func(context.Context, *sqlx.DB) (repository.DedupeStats, error) {
					return repository.DedupeStats{Messages: 2, Chunks: 1}, nil
				},
				dedupeFn: func(context.Context, *sqlx.DB) (repository.DedupeStats, error) {
					called = true
					return repository.DedupeStats{}, nil
				},
			}
		}
		var buf bytes.Buffer
		res, err := Run(t.Context(), nil, Options{Report: &buf, Database: "db"})
		require.NoError(t, err)
		assert.Equal(t, int64(2), res.Counts.Messages)
		assert.False(t, called)
		assert.Contains(t, buf.String(), "Duplicate messages: 2")
		assert.Contains(t, buf.String(), "Run with -execute to perform dedupe.")
	})

	t.Run("execute with report", func(t *testing.T) {
		newRepo = func(repository.MessageDedupeMode) repository.DedupeRepository {
			return stubRepo{
				previewFn: func(context.Context, *sqlx.DB) (repository.DedupeStats, error) {
					return repository.DedupeStats{Messages: 1}, nil
				},
				dedupeFn: func(context.Context, *sqlx.DB) (repository.DedupeStats, error) {
					return repository.DedupeStats{Messages: 1}, nil
				},
			}
		}
		var buf bytes.Buffer
		res, err := Run(t.Context(), nil, Options{Execute: true, Report: &buf, Database: "db"})
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.Removed.Messages)
		assert.Contains(t, buf.String(), "Removed messages: 1")
	})

	t.Run("log only mode does not write report", func(t *testing.T) {
		newRepo = func(repository.MessageDedupeMode) repository.DedupeRepository {
			return stubRepo{
				previewFn: func(context.Context, *sqlx.DB) (repository.DedupeStats, error) {
					return repository.DedupeStats{}, nil
				},
				dedupeFn: func(context.Context, *sqlx.DB) (repository.DedupeStats, error) {
					return repository.DedupeStats{}, nil
				},
			}
		}
		res, err := Run(t.Context(), nil, Options{Database: "db"})
		require.NoError(t, err)
		assert.Equal(t, repository.DedupeStats{}, res.Counts)
	})

	t.Run("execute returns dedupe error", func(t *testing.T) {
		newRepo = func(repository.MessageDedupeMode) repository.DedupeRepository {
			return stubRepo{
				previewFn: func(context.Context, *sqlx.DB) (repository.DedupeStats, error) {
					return repository.DedupeStats{}, nil
				},
				dedupeFn: func(context.Context, *sqlx.DB) (repository.DedupeStats, error) {
					return repository.DedupeStats{}, errors.New("boom")
				},
			}
		}
		_, err := Run(t.Context(), nil, Options{Execute: true, Database: "db"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "deduplicate entities")
	})

	t.Run("passes mode to repository", func(t *testing.T) {
		var got repository.MessageDedupeMode
		newRepo = func(mode repository.MessageDedupeMode) repository.DedupeRepository {
			got = mode
			return stubRepo{
				previewFn: func(context.Context, *sqlx.DB) (repository.DedupeStats, error) {
					return repository.DedupeStats{}, nil
				},
				dedupeFn: func(context.Context, *sqlx.DB) (repository.DedupeStats, error) {
					return repository.DedupeStats{}, nil
				},
			}
		}
		_, err := Run(t.Context(), nil, Options{Mode: repository.MessageDedupeKey})
		require.NoError(t, err)
		assert.Equal(t, repository.MessageDedupeKey, got)
	})
}
