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

// Package source provides archive readers for different output formats.
//
// Currently, the following formats are supported:
//   - archive
//   - database
//   - dump
//   - Slack Export
//
// One should use [Load] function to load the source from the file system.  It
// will automatically detect the format and return the appropriate reader.
//
// All sources implement the [Sourcer] interface, which provides methods to
// retrieve data from the source.  The [Resumer] interface is implemented by
// sources that can be resumed.  The [SourceResumeCloser] interface is
// implemented by sources that can be resumed and closed.
//
// There are two distinct error types of interest, [ErrNotFound] and
// [ErrNotSupported].  See their documentation for more information.
//
// # What is a Source?
//
// A source is a generic interface over the data source.  It can be implemented
// not only around Slack data, but also around any other messenger data, as it
// represents common entities every messenger has, for example "message",
// "thread", "channel", for Telegram would be "message", "reply", "chat"
// respectively.  The caveat is that non-Slack source's entities would need to
// be converted to Slack entities, which shouldn't be hard to do, unless you're
// aiming to replicate formatting.
//
// In this package, for now, the source is implemented only for data
// originating from Slackdump and Slack.
//
// # Loading a Source
//
// If you know what source you are loading, or need  a particular source type
// you can use a concrete Open* function, such as [OpenDatabase].  If you
// don't, you can use [Load] function that will determine the source type and
// return the appropriate source.
//
// It is a good idea to defer the closing of the source, as it may be a
// database connection or a file handle.  The [SourceResumeCloser] interface,
// returned by [Load] implements [io.Closer] interface, so you can use it with
// defer statement.  For example:
//
//	src, err := source.Load(ctx, "path/to/source")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer src.Close()
//
// Within your code, you can call the Type method which will return the source
// type.
//
// # Source Types
//
// The source type returned by [Type] is a bitmask, you can use [Has] method to
// check if particular flag is set.  Flag constants all start with F* and have
// [Flags] type.
//
// The source type returned by Type method on a particular source will only
// have the type flag set, without any additional flags (as of v3.1.0, but this
// may change in future versions, so always use Has method to be on the safe
// side).
package source
