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
package testutil

import (
	"iter"
	"testing"

	"github.com/stretchr/testify/assert"
)

type IterVal[T, U any] struct {
	T T
	U U
}

func Seq2Collect[T, U any](it iter.Seq2[T, U]) (ret []IterVal[T, U]) {
	for t, u := range it {
		ret = append(ret, IterVal[T, U]{T: t, U: u})
	}
	return
}

func IterVal2Iter[T, U any](s []IterVal[T, U]) iter.Seq2[T, U] {
	return func(yield func(T, U) bool) {
		for _, v := range s {
			if !yield(v.T, v.U) {
				return
			}
		}
	}
}

func Slice2Seq2[T any](s []T) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for _, v := range s {
			if !yield(v, nil) {
				return
			}
		}
	}
}

// TestResult is a pair of value and error to use in the test iterators.
type TestResult[T any] struct {
	V   T
	Err error
}

func ToTestResult[T any](v T, err error) TestResult[T] {
	return TestResult[T]{V: v, Err: err}
}

// ToIter converts a slice of testResult to an iter.Seq2.
func ToIter[T any](v []TestResult[T]) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for _, r := range v {
			if !yield(r.V, r.Err) {
				break
			}
		}
	}
}

// Collect collects the values from the iterator into a slice of TestResult.
func Collect[T any](t *testing.T, it iter.Seq2[T, error]) []TestResult[T] {
	t.Helper()
	var ret []TestResult[T]
	for v, err := range it {
		ret = append(ret, TestResult[T]{v, err})
	}
	return ret
}

// AssertIterResult checks if the iterator returns the expected values.
func AssertIterResult[T any](t *testing.T, want []TestResult[T], got iter.Seq2[T, error]) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic, possibly different number of results: %v", r)
		}
	}()

	var i int
	for v, err := range got {
		assert.Equalf(t, want[i].V, v, "value %d", i)
		if (err != nil) != (want[i].Err != nil) {
			t.Errorf("got error on %d %v, want %v", i, err, want[i].Err)
		}
		i++
	}
	if i != len(want) {
		t.Errorf("got %d results, want %d", i, len(want))
	}
}

// SliceToTestResult converts a slice of values to a slice of TestResult.
func SliceToTestResult[E any, T []E](t T) []TestResult[E] {
	r := make([]TestResult[E], len(t))
	for i, v := range t {
		r[i].V = v
	}
	return r
}
