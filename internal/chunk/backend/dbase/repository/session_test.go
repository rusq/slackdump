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
package repository

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_sessionRepository_Insert(t *testing.T) {
	type args struct {
		ctx  context.Context
		conn PrepareExtContext
		s    *Session
	}
	tests := []struct {
		name    string
		r       sessionRepository
		args    args
		prepFn  utilityFn
		want    int64
		wantErr bool
		checkFn utilityFn
	}{
		{
			name: "inserts new empty session",
			args: args{
				ctx:  t.Context(),
				conn: testConn(t),
				s:    &Session{},
			},
			want:    1,
			checkFn: checkCount("session", 1),
		},
		{
			name: "fails if parent does not exist",
			args: args{
				ctx:  t.Context(),
				conn: testConn(t),
				s: &Session{
					ParentID: ptr[int64](1),
				},
			},
			wantErr: true,
			checkFn: checkCount("session", 0),
		},
		{
			name: "inserts new session with parent",
			args: args{
				ctx:  t.Context(),
				conn: testConn(t),
				s: &Session{
					ParentID: ptr[int64](1),
					Mode:     "test",
					Args:     "args",
				},
			},
			prepFn: func(t *testing.T, conn PrepareExtContext) {
				if _, err := conn.ExecContext(t.Context(), "INSERT INTO session (id,mode,args) VALUES (1,'test','args')"); err != nil {
					t.Fatalf("err = %v; want nil", err)
				}
			},
			want:    2,
			checkFn: checkCount("session", 2),
		},
		{
			name: "all fields",
			args: args{
				ctx:  t.Context(),
				conn: testConn(t),
				s: &Session{
					ID:             10,
					CreatedAt:      time.Date(2009, time.September, 16, 5, 6, 7, 0, time.UTC),
					FromTS:         ptr(time.Date(2010, time.September, 16, 5, 6, 7, 0, time.UTC)),
					ToTS:           ptr(time.Date(2010, time.September, 16, 5, 6, 7, 0, time.UTC)),
					Finished:       true,
					FilesEnabled:   true,
					AvatarsEnabled: true,
					Mode:           "test",
					Args:           "arg1 arg2 arg3",
				},
			},
			want:    10,
			checkFn: checkCount("session", 1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.conn)
			}
			r := sessionRepository{}
			got, err := r.Insert(tt.args.ctx, tt.args.conn, tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("sessionRepository.Insert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("sessionRepository.Insert() = %v, want %v", got, tt.want)
			}
			if tt.checkFn != nil {
				tt.checkFn(t, tt.args.conn)
			}
		})
	}
}

func Test_sessionRepository_Finish(t *testing.T) {
	type args struct {
		ctx  context.Context
		conn PrepareExtContext
		id   int64
	}
	tests := []struct {
		name    string
		r       sessionRepository
		args    args
		prepFn  utilityFn
		want    int64
		wantErr bool
		checkFn utilityFn
	}{
		{
			name: "finishes existing session",
			args: args{
				ctx:  t.Context(),
				conn: testConn(t),
				id:   1,
			},
			prepFn: func(t *testing.T, conn PrepareExtContext) {
				r := sessionRepository{}
				if _, err := r.Insert(t.Context(), conn, &Session{}); err != nil {
					t.Fatalf("err = %v; want nil", err)
				}
			},
			want: 1,
			checkFn: func(t *testing.T, conn PrepareExtContext) {
				var finished bool
				if err := conn.QueryRowxContext(t.Context(), "SELECT finished FROM session WHERE id = 1").Scan(&finished); err != nil {
					t.Fatalf("err = %v; want nil", err)
				}
				if !finished {
					t.Errorf("finished = %v; want true", finished)
				}
			},
		},
		{
			name: "fails if session does not exist",
			args: args{
				ctx:  t.Context(),
				conn: testConn(t),
				id:   1,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.conn)
			}
			r := sessionRepository{}
			got, err := r.Finalise(tt.args.ctx, tt.args.conn, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("sessionRepository.Finish() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("sessionRepository.Finish() = %v, want %v", got, tt.want)
			}
			if tt.checkFn != nil {
				tt.checkFn(t, tt.args.conn)
			}
		})
	}
}

func Test_sessionRepository_Get(t *testing.T) {
	type args struct {
		ctx  context.Context
		conn PrepareExtContext
		id   int64
	}
	tests := []struct {
		name    string
		r       sessionRepository
		prepFn  utilityFn
		args    args
		want    *Session
		wantErr bool
	}{
		{
			name: "gets existing session",
			r:    sessionRepository{},
			prepFn: func(t *testing.T, conn PrepareExtContext) {
				r := sessionRepository{}
				testSession := &Session{
					CreatedAt:      time.Date(2009, time.September, 16, 5, 6, 7, 0, time.UTC),
					UpdatedAt:      time.Date(2009, time.September, 16, 5, 6, 7, 0, time.UTC),
					FromTS:         ptr(time.Date(2010, time.September, 16, 5, 6, 7, 0, time.UTC)),
					ToTS:           ptr(time.Date(2010, time.September, 16, 5, 6, 7, 0, time.UTC)),
					Finished:       true,
					FilesEnabled:   true,
					AvatarsEnabled: true,
					Mode:           "test",
					Args:           "arg1 arg2 arg3",
				}
				if _, err := r.Insert(t.Context(), conn, testSession); err != nil {
					t.Fatalf("err = %v; want nil", err)
				}
			},
			args: args{ctx: t.Context(), conn: testConn(t), id: 1},
			want: &Session{
				ID:             1,
				CreatedAt:      time.Date(2009, time.September, 16, 5, 6, 7, 0, time.UTC),
				UpdatedAt:      time.Date(2009, time.September, 16, 5, 6, 7, 0, time.UTC),
				FromTS:         ptr(time.Date(2010, time.September, 16, 5, 6, 7, 0, time.UTC)),
				ToTS:           ptr(time.Date(2010, time.September, 16, 5, 6, 7, 0, time.UTC)),
				Finished:       true,
				FilesEnabled:   true,
				AvatarsEnabled: true,
				Mode:           "test",
				Args:           "arg1 arg2 arg3",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.conn)
			}
			r := sessionRepository{}
			got, err := r.Get(tt.args.ctx, tt.args.conn, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("sessionRepository.Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_sessionRepository_Update(t *testing.T) {
	type args struct {
		ctx  context.Context
		conn PrepareExtContext
		s    *Session
	}
	tests := []struct {
		name    string
		r       sessionRepository
		args    args
		prepFn  utilityFn
		want    int64
		wantErr bool
		checkFn utilityFn
	}{
		{
			name: "updates existing session",
			args: args{
				ctx:  t.Context(),
				conn: testConn(t),
				s: &Session{
					ID:   1,
					Mode: "resume",
				},
			},
			prepFn: func(t *testing.T, conn PrepareExtContext) {
				r := sessionRepository{}
				if _, err := r.Insert(t.Context(), conn, &Session{Mode: "archive"}); err != nil {
					t.Fatalf("err = %v; want nil", err)
				}
			},
			want: 1,
			checkFn: func(t *testing.T, conn PrepareExtContext) {
				var mode string
				if err := conn.QueryRowxContext(t.Context(), "SELECT mode FROM session WHERE id = 1").Scan(&mode); err != nil {
					t.Fatalf("err = %v; want nil", err)
				}
				if mode != "resume" {
					t.Errorf("mode = %v; want resume", mode)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.conn)
			}
			r := sessionRepository{}
			got, err := r.Update(tt.args.ctx, tt.args.conn, tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("sessionRepository.Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("sessionRepository.Update() = %v, want %v", got, tt.want)
			}
			if tt.checkFn != nil {
				tt.checkFn(t, tt.args.conn)
			}
		})
	}
}

func Test_sessionRepository_Last(t *testing.T) {
	var (
		testSess1 = &Session{
			ID:             1,
			CreatedAt:      time.Date(2009, time.September, 16, 5, 6, 7, 0, time.UTC),
			UpdatedAt:      time.Date(2009, time.September, 16, 5, 6, 7, 0, time.UTC),
			Finished:       true,
			FilesEnabled:   false,
			AvatarsEnabled: false,
			Mode:           "a",
			Args:           "b",
		}
		testSess2 = &Session{
			ID:             2,
			ParentID:       ptr(int64(1)),
			CreatedAt:      time.Date(2009, time.September, 17, 5, 6, 7, 0, time.UTC),
			UpdatedAt:      time.Date(2009, time.September, 18, 5, 6, 7, 0, time.UTC),
			Finished:       false,
			FilesEnabled:   true,
			AvatarsEnabled: false,
			Mode:           "c",
			Args:           "d",
		}
	)
	twoSess := func(t *testing.T, conn PrepareExtContext) {
		r := sessionRepository{}
		if _, err := r.Insert(t.Context(), conn, testSess1); err != nil {
			t.Fatalf("err = %v; want nil", err)
		}
		if _, err := r.Insert(t.Context(), conn, testSess2); err != nil {
			t.Fatalf("err = %v; want nil", err)
		}
	}
	type args struct {
		ctx      context.Context
		conn     PrepareExtContext
		finished *bool
	}
	tests := []struct {
		name    string
		r       sessionRepository
		args    args
		prepFn  utilityFn
		want    *Session
		wantErr bool
	}{
		{
			name:    "gets last session",
			r:       sessionRepository{},
			args:    args{ctx: t.Context(), conn: testConn(t), finished: nil},
			prepFn:  twoSess,
			want:    testSess2,
			wantErr: false,
		},
		{
			name:   "gets last finished session",
			r:      sessionRepository{},
			args:   args{ctx: t.Context(), conn: testConn(t), finished: ptr(true)},
			prepFn: twoSess,
			want:   testSess1,
		},
		{
			name:   "gets last unfinished session",
			r:      sessionRepository{},
			args:   args{ctx: t.Context(), conn: testConn(t), finished: ptr(false)},
			prepFn: twoSess,
			want:   testSess2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.conn)
			}
			r := sessionRepository{}
			got, err := r.Last(tt.args.ctx, tt.args.conn, tt.args.finished)
			if (err != nil) != tt.wantErr {
				t.Errorf("sessionRepository.Last() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sessionRepository.Last() = %v, want %v", got, tt.want)
			}
		})
	}
}
