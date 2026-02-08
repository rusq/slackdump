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
package structures

// in this file: slack timestamp parsing functions

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseThreadID parses the thread id (ie. p1577694990000400) and returns
// time.Time.
func ParseThreadID(threadID string) (time.Time, error) {
	if len(threadID) == 0 || threadID[0] != 'p' {
		return time.Time{}, errors.New("not a thread ID")
	}
	if _, err := strconv.ParseInt(threadID[1:], 10, 64); err != nil {
		return time.Time{}, errors.New("invalid thread ID")
	}
	return ParseSlackTS(threadID[1:11] + "." + threadID[11:])
}

// ParseSlackTS parses the slack timestamp.
func ParseSlackTS(timestamp string) (time.Time, error) {
	const (
		base = 10
		bit  = 64
	)
	sSec, sMicro, found := strings.Cut(timestamp, ".")
	if sSec == "" {
		return time.Time{}, errors.New("empty timestamp")
	}
	var t int64
	var err error
	if !found {
		t, err = strconv.ParseInt(sSec+"000000", base, bit)
	} else {
		t, err = strconv.ParseInt(sSec+sMicro, base, bit)
	}
	if err != nil {
		return time.Time{}, err
	}
	return time.UnixMicro(t).UTC(), nil
}

func FormatSlackTS(ts time.Time) string {
	if ts.IsZero() || ts.Before(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)) {
		return ""
	}
	hi := ts.Unix()
	lo := ts.UnixMicro() % 1_000_000
	return fmt.Sprintf("%d.%06d", hi, lo)
}

func ThreadIDtoTS(threadID string) string {
	if len(threadID) == 0 || threadID[0] != 'p' {
		return ""
	}
	if _, err := strconv.ParseInt(threadID[1:], 10, 64); err != nil {
		return ""
	}
	return threadID[1:11] + "." + threadID[11:]
}
