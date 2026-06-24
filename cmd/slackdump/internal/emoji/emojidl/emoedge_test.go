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

package emojidl

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v4/internal/edge"
)

type edgeEmojiListerFunc func(context.Context) iter.Seq2[edge.EmojiResult, error]

func (f edgeEmojiListerFunc) AdminEmojiList(ctx context.Context) iter.Seq2[edge.EmojiResult, error] {
	return f(ctx)
}

func runDlEdgeFS(t *testing.T, ctx context.Context, sess EdgeEmojiLister, fsa fsadapter.FS, opt *Options) error {
	t.Helper()

	done := make(chan error, 1)
	go func() {
		done <- DlEdgeFS(ctx, sess, fsa, opt, nil)
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(time.Second):
		t.Fatal("DlEdgeFS timed out")
		return nil
	}
}

func TestDlEdgeFS(t *testing.T) {
	t.Run("generator error returns promptly", func(t *testing.T) {
		wantErr := errors.New("list failed")
		sess := edgeEmojiListerFunc(func(context.Context) iter.Seq2[edge.EmojiResult, error] {
			return func(yield func(edge.EmojiResult, error) bool) {
				yield(edge.EmojiResult{}, wantErr)
			}
		})
		fsa, err := fsadapter.New(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		defer fsa.Close()

		if err := runDlEdgeFS(t, t.Context(), sess, fsa, nil); !errors.Is(err, wantErr) {
			t.Fatalf("DlEdgeFS() error = %v, want %v", err, wantErr)
		}
	})

	t.Run("fail fast worker error returns promptly", func(t *testing.T) {
		wantErr := errors.New("fetch failed")
		setGlobalFetchFn(func(context.Context, fsadapter.FS, string, string, string) error {
			return wantErr
		})
		defer setGlobalFetchFn(emptyFetchFn)

		sess := edgeEmojiListerFunc(func(context.Context) iter.Seq2[edge.EmojiResult, error] {
			return func(yield func(edge.EmojiResult, error) bool) {
				yield(edge.EmojiResult{
					Total: 2,
					Emoji: []edge.Emoji{
						{Name: "one", URL: "https://example.com/one.png"},
						{Name: "two", URL: "https://example.com/two.png"},
					},
				}, nil)
			}
		})
		fsa, err := fsadapter.New(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		defer fsa.Close()

		err = runDlEdgeFS(t, t.Context(), sess, fsa, &Options{FailFast: true, WithFiles: true})
		if !errors.Is(err, wantErr) {
			t.Fatalf("DlEdgeFS() error = %v, want %v", err, wantErr)
		}
	})

	t.Run("canceled context returns promptly", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		sess := edgeEmojiListerFunc(func(context.Context) iter.Seq2[edge.EmojiResult, error] {
			return func(yield func(edge.EmojiResult, error) bool) {
				yield(edge.EmojiResult{}, context.Canceled)
			}
		})
		fsa, err := fsadapter.New(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		defer fsa.Close()

		if err := runDlEdgeFS(t, ctx, sess, fsa, nil); !errors.Is(err, context.Canceled) {
			t.Fatalf("DlEdgeFS() error = %v, want %v", err, context.Canceled)
		}
	})

	t.Run("success writes index", func(t *testing.T) {
		dir := t.TempDir()
		sess := edgeEmojiListerFunc(func(context.Context) iter.Seq2[edge.EmojiResult, error] {
			return func(yield func(edge.EmojiResult, error) bool) {
				yield(edge.EmojiResult{
					Total: 1,
					Emoji: []edge.Emoji{{Name: "party", URL: "https://example.com/party.png"}},
				}, nil)
			}
		})
		fsa, err := fsadapter.New(dir)
		if err != nil {
			t.Fatal(err)
		}
		defer fsa.Close()

		if err := runDlEdgeFS(t, t.Context(), sess, fsa, &Options{WithFiles: false}); err != nil {
			t.Fatalf("DlEdgeFS() error = %v", err)
		}
		b, err := os.ReadFile(filepath.Join(dir, "index.json"))
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]edge.Emoji
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatal(err)
		}
		if _, ok := got["party"]; !ok {
			t.Fatalf("index missing emoji: %v", got)
		}
	})

	t.Run("alias only completes without fetch", func(t *testing.T) {
		setGlobalFetchFn(func(context.Context, fsadapter.FS, string, string, string) error {
			t.Fatal("fetchFn called for alias")
			return nil
		})
		defer setGlobalFetchFn(emptyFetchFn)

		sess := edgeEmojiListerFunc(func(context.Context) iter.Seq2[edge.EmojiResult, error] {
			return func(yield func(edge.EmojiResult, error) bool) {
				yield(edge.EmojiResult{
					Total: 1,
					Emoji: []edge.Emoji{{Name: "shipit", IsAlias: 1, AliasFor: "squirrel"}},
				}, nil)
			}
		})
		fsa, err := fsadapter.New(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		defer fsa.Close()

		if err := runDlEdgeFS(t, t.Context(), sess, fsa, &Options{WithFiles: true}); err != nil {
			t.Fatalf("DlEdgeFS() error = %v", err)
		}
	})
}
