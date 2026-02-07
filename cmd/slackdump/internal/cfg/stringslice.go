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
package cfg

import (
	"flag"
	"fmt"
	"slices"
	"strings"

	"github.com/rusq/slackdump/v3"
)

const stringSliceSep = ","

// StringSlice provides a flag.Value interface for a slice of strings.
type StringSlice []string

var _ flag.Value = new(StringSlice)

func (ss *StringSlice) Set(s string) error {
	parts := strings.Split(s, stringSliceSep)
	*ss = parts
	return nil
}

func (ss *StringSlice) String() string {
	return strings.Join(*ss, stringSliceSep)
}

type slackChanTypes StringSlice

func (ss *slackChanTypes) Set(s string) error {
	(*StringSlice)(ss).Set(s)
	for _, v := range *ss {
		if !slices.Contains(slackdump.AllChanTypes, v) {
			return fmt.Errorf("allowed values are: %v", slackdump.AllChanTypes)
		}
	}
	return nil
}

func (ss *slackChanTypes) String() string {
	return (*StringSlice)(ss).String()
}
