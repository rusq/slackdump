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

package transform

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/rusq/slack"
)

type exportCoordinatorTestConverter struct {
	mu      sync.Mutex
	users   []slack.User
	convert func(context.Context, string, string) error
}

func (c *exportCoordinatorTestConverter) Convert(ctx context.Context, channelID, threadID string) error {
	if c.convert == nil {
		return nil
	}
	return c.convert(ctx, channelID, threadID)
}

func (c *exportCoordinatorTestConverter) SetUsers(users []slack.User) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.users = users
}

func (c *exportCoordinatorTestConverter) HasUsers() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.users) > 0
}

func testUsers() []slack.User {
	return []slack.User{{ID: "U123", Name: "test"}}
}

func TestExportCoordinator_Start(t *testing.T) {
	t.Run("after close returns error", func(t *testing.T) {
		cvt := &exportCoordinatorTestConverter{}
		ec := NewExportCoordinator(t.Context(), cvt, WithUsers(testUsers()))

		if err := ec.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		if err := ec.Start(t.Context()); err == nil {
			t.Fatal("Start() error = nil, want error")
		}
	})

	t.Run("concurrent close does not panic", func(t *testing.T) {
		for range 100 {
			cvt := &exportCoordinatorTestConverter{}
			ec := NewExportCoordinator(t.Context(), cvt, WithUsers(testUsers()))

			done := make(chan struct{})
			go func() {
				defer close(done)
				var wg sync.WaitGroup
				wg.Add(2)
				go func() {
					defer wg.Done()
					_ = ec.Start(t.Context())
				}()
				go func() {
					defer wg.Done()
					_ = ec.Close()
				}()
				wg.Wait()
			}()

			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("concurrent Start and Close timed out")
			}
		}
	})

	t.Run("repeated start does not block", func(t *testing.T) {
		cvt := &exportCoordinatorTestConverter{}
		ec := NewExportCoordinator(t.Context(), cvt, WithUsers(testUsers()))
		defer ec.Close()

		for range 3 {
			done := make(chan error, 1)
			go func() {
				done <- ec.Start(t.Context())
			}()
			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("Start() error = %v", err)
				}
			case <-time.After(time.Second):
				t.Fatal("Start() timed out")
			}
		}
	})
}

func TestExportCoordinator_Transform(t *testing.T) {
	t.Run("after close returns ErrClosed", func(t *testing.T) {
		cvt := &exportCoordinatorTestConverter{}
		ec := NewExportCoordinator(t.Context(), cvt, WithUsers(testUsers()))

		if err := ec.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		if err := ec.Transform(t.Context(), "C123", ""); !errors.Is(err, ErrClosed) {
			t.Fatalf("Transform() error = %v, want %v", err, ErrClosed)
		}
	})
}

func TestExportCoordinator_Close(t *testing.T) {
	t.Run("returns convert error", func(t *testing.T) {
		wantErr := errors.New("convert failed")
		cvt := &exportCoordinatorTestConverter{
			convert: func(context.Context, string, string) error {
				return wantErr
			},
		}
		ec := NewExportCoordinator(t.Context(), cvt, WithUsers(testUsers()))

		if err := ec.Start(t.Context()); err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		if err := ec.Transform(t.Context(), "C123", ""); err != nil {
			t.Fatalf("Transform() error = %v", err)
		}
		if err := ec.Close(); !errors.Is(err, wantErr) {
			t.Fatalf("Close() error = %v, want %v", err, wantErr)
		}
	})
}
