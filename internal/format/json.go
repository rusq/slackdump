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
package format

import (
	"context"
	"encoding/json"
	"io"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/types"
)

type jsonOptions struct {
	prefix string
	indent string
}

// JSON is the json formatter.
type JSON struct {
	opts jsonOptions
}

func init() {
	converters[CJSON] = NewJSON
}

func NewJSON(opts ...Option) Formatter {
	settings := options{
		jsonOptions: jsonOptions{},
	}
	for _, fn := range opts {
		fn(&settings)
	}
	return &JSON{opts: settings.jsonOptions}
}

// Extension returns the file extension for the formatter.
func (j JSON) Extension() string {
	return ".json"
}

func JSONPrefix(prefix string) Option {
	return func(o *options) {
		o.jsonOptions.prefix = prefix
	}
}

func JSONIndent(indent string) Option {
	return func(o *options) {
		o.jsonOptions.indent = indent
	}
}

func (j JSON) Conversation(ctx context.Context, w io.Writer, u []slack.User, conv *types.Conversation) error {
	return j.enc(w).Encode(conv)
}

func (j JSON) Channels(ctx context.Context, w io.Writer, u []slack.User, chans []slack.Channel) error {
	return j.enc(w).Encode(chans)
}

func (j JSON) Users(ctx context.Context, w io.Writer, u []slack.User) error {
	return j.enc(w).Encode(u)
}

func (j JSON) enc(w io.Writer) *json.Encoder {
	enc := json.NewEncoder(w)
	enc.SetIndent(j.opts.prefix, j.opts.indent)
	return enc
}
