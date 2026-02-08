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
// Package format provides formatting functions for different output format
// types.
package format

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/types"
)

// Type is the converter type.
//
//go:generate stringer -type Type -trimprefix C format.go
type Type int

const (
	CUnknown Type = iota // Unknown converter type
	CText                // CText is the plain text converter
	CCSV                 // CCSV is the CSV converter
	CJSON                // CJSON is JSON format converter
)

var Descriptions = map[Type]string{
	CText: "Plain text format",
	CCSV:  "CSV format",
	CJSON: "JSON format",
}

// Types is a list of converter types.
type Types []Type

func (tt Types) String() string {
	var s []string
	for _, t := range tt {
		s = append(s, t.String())
	}
	return strings.Join(s, ", ")
}

func All() Types {
	return slices.SortedFunc(maps.Keys(converters), func(a, b Type) int {
		return cmp.Compare(a.String(), b.String())
	})
}

// Formatter is a converter interface that each formatter must implement.
type Formatter interface {
	// Conversation writes the conversation to the writer.
	Conversation(ctx context.Context, w io.Writer, u []slack.User, conv *types.Conversation) error
	// Channels writes the channel list to the writer.
	Channels(ctx context.Context, w io.Writer, u []slack.User, chans []slack.Channel) error
	// Users writes the user list to the writer.
	Users(ctx context.Context, w io.Writer, u []slack.User) error
	// Extension returns the file extension for the formatter.
	Extension() string
}

type options struct {
	textOptions
	csvOptions
	jsonOptions
	bare bool // bare output format
}

// Option is the converter option.
type Option func(*options)

var converters = make(map[Type]func(opts ...Option) Formatter)

func (e *Type) Set(v string) error {
	v = strings.ToLower(v)
	for i := 0; i < len(_Type_index)-1; i++ {
		if strings.ToLower(_Type_name[_Type_index[i]:_Type_index[i+1]]) == v {
			*e = Type(i)
			return nil
		}
	}
	return fmt.Errorf("unknown converter: %s", v)
}

func (e *Type) FormatFunc() (func(opts ...Option) Formatter, bool) {
	fn, ok := converters[*e]
	return fn, ok
}

// WithBareFormat allows to set the bare output format for the formatter that
// supports it.
func WithBareFormat(b bool) Option {
	return func(o *options) {
		o.bare = b
	}
}

// userReplacer returns a replacer that replaces all user IDs with their
// DisplayNames.
func userReplacer(userIdx structures.UserIndex) *strings.Replacer {
	if len(userIdx) == 0 {
		return strings.NewReplacer()
	}
	replacements := make([]string, 0, len(userIdx)*2)
	for k := range userIdx {
		replacements = append(replacements, k, userIdx.DisplayName(k))
	}
	return strings.NewReplacer(replacements...)
}
