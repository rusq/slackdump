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

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	// excludePrefix is the prefix that is used to mark channel
	// exclusions, i.e. for export or when downloading conversations.
	excludePrefix = "^"
	filePrefix    = "@"
	timeSeparator = ","
	TimeLayout    = "2006-01-02T15:04:05"
	DateLayout    = time.DateOnly

	// maxFileEntries is the maximum non-empty entries that will be read
	// from the file.
	maxFileEntries = 1048576
)

var (
	ErrMaxFileSize = errors.New("maximum file size exceeded")
	ErrEmptyList   = errors.New("empty list")
)

type EntityItem struct {
	Id      string
	Oldest  time.Time
	Latest  time.Time
	Include bool
}

func (ei *EntityItem) String() string {
	var sb strings.Builder
	if !ei.Include {
		sb.WriteString(excludePrefix)
	}
	sb.WriteString(ei.Id)
	if !ei.Oldest.IsZero() {
		sb.WriteString(timeSeparator)
		sb.WriteString(ei.Oldest.Format(TimeLayout))
	}
	if !ei.Latest.IsZero() {
		sb.WriteString(timeSeparator)
		sb.WriteString(ei.Latest.Format(TimeLayout))
	}
	return sb.String()
}

// EntityList is an Inclusion/Exclusion list
type EntityList struct {
	index               map[string]*EntityItem
	mu                  sync.RWMutex
	hasIncludes         bool
	hasExcludes         bool
	hasExplicitIncludes bool
}

func (el *EntityList) IncludeCount() int {
	el.mu.RLock()
	defer el.mu.RUnlock()
	var count int
	for _, item := range el.index {
		if item.Include {
			count++
		}
	}
	return count
}

func (el *EntityList) ExcludeCount() int {
	el.mu.RLock()
	defer el.mu.RUnlock()
	var count int
	for _, item := range el.index {
		if !item.Include {
			count++
		}
	}
	return count
}

func HasExcludePrefix(s string) bool {
	return strings.HasPrefix(s, excludePrefix)
}

func hasFilePrefix(s string) bool {
	return strings.HasPrefix(s, filePrefix)
}

// Overlay overlays the other EntityList on the el, overriding entities.  Thus,
// other EntityList will have a priority over el.
func (el *EntityList) Overlay(other *EntityList) {
	if other == nil {
		return
	}
	if el == other {
		return
	}
	el.mu.Lock()
	defer el.mu.Unlock()
	other.mu.RLock()
	defer other.mu.RUnlock()

	el.hasExplicitIncludes = el.hasExplicitIncludes || other.hasExplicitIncludes
	if len(other.index) == 0 {
		return
	}

	maps.Copy(el.index, other.index)
	el.updateEffectiveFlags()
}

// NewEntityList creates an EntityList from a slice of IDs or URLs (entities).
func NewEntityList(entities []string) (*EntityList, error) {
	entries, err := buildEntryIndex(entities)
	if err != nil {
		return nil, err
	}
	el := EntityList{hasExplicitIncludes: entries.hasExplicitIncludes}
	el.fromIndex(entries.index)

	return &el, nil
}

// NewEntityListFromString creates an EntityList from a space-separated list of
// entities.
func NewEntityListFromString(s string) (*EntityList, error) {
	if len(s) == 0 {
		return nil, ErrEmptyList
	}
	ee := strings.Split(s, " ")
	if len(ee) == 0 {
		return nil, ErrEmptyList
	}
	return NewEntityList(ee)
}

// NewEntityListFromItems creates an EntityList from items.  As with
// [NewEntityList], an exclusion takes precedence over an include with the same
// ID regardless of order; otherwise, a later item replaces an earlier one.
func NewEntityListFromItems(items ...EntityItem) *EntityList {
	el := EntityList{
		index: make(map[string]*EntityItem, len(items)),
	}
	for _, item := range items {
		el.hasExplicitIncludes = el.hasExplicitIncludes || item.Include
		if existing := el.index[item.Id]; item.Include && existing != nil && !existing.Include {
			continue // exclusions take precedence
		}
		el.index[item.Id] = &item
	}
	el.updateEffectiveFlags()
	return &el
}

// ValidateEntityList validates a space-separated list of entities.
func ValidateEntityList(s string) error {
	_, err := NewEntityListFromString(s)
	return err
}

// SplitEntryList splits the string by space.
func SplitEntryList(s string) []string {
	return strings.Split(s, " ")
}

// loadEntityIndex loads the entries from a file.
func loadEntityIndex(filename string) (entryIndex, error) {
	f, err := os.Open(filename)
	if err != nil {
		return entryIndex{}, err
	}
	defer f.Close()
	return readEntityIndex(f, maxFileEntries)
}

// readEntityIndex is a rather naïve implementation that reads the entire file up
// to maxEntries entities (empty lines are skipped), and populates the slice of
// strings, which is then passed to buildEntryIndex. On large lists it will
// probably use a silly amount of memory.
func readEntityIndex(r io.Reader, maxEntries int) (entryIndex, error) {
	br := bufio.NewReader(r)
	var elements []string
	total := 0
	var exit bool
	for n := 1; ; n++ {
		if total >= maxEntries {
			return entryIndex{}, fmt.Errorf("%w (%d)", ErrMaxFileSize, maxFileEntries)
		}
		line, err := br.ReadString('\n')
		if errors.Is(err, io.EOF) {
			exit = true
		} else if err != nil {
			return entryIndex{}, err
		}
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			if exit {
				break
			}
			continue
		}
		// test if it's a valid line
		elements = append(elements, line)
		if exit {
			break
		}

		total++
	}
	return buildEntryIndex(elements)
}

func getTimeTuple(item string) []string {
	if strings.HasPrefix(item, filePrefix) {
		return []string{item}
	}
	return strings.SplitN(item, timeSeparator, 3)
}

func (el *EntityList) fromIndex(index map[string]bool) {
	el.index = make(map[string]*EntityItem, len(index))

	el.mu.Lock()
	defer el.mu.Unlock()

	for ent, include := range index {
		parts := getTimeTuple(ent)

		// Exclusions take precedence even when the date bounds differ.
		if existing := el.index[parts[0]]; include && existing != nil && !existing.Include {
			continue
		}

		item := &EntityItem{
			Id:      parts[0],
			Include: include,
		}
		if len(parts) > 1 {
			item.Oldest, _ = TimeParse(parts[1])
		}
		if len(parts) == 3 {
			item.Latest, _ = TimeParse(parts[2])
		}

		el.index[item.Id] = item
	}

	el.updateEffectiveFlags()
}

// updateEffectiveFlags assumes el.mu is held or el is being constructed.
func (el *EntityList) updateEffectiveFlags() {
	el.hasIncludes = false
	el.hasExcludes = false
	for _, item := range el.index {
		if item.Include {
			el.hasIncludes = true
		} else {
			el.hasExcludes = true
		}
	}
}

// TimeParse parses a string that can be either a date in 2006-01-02 layout or
// time in 2006-01-02T15:04:05 layout.
func TimeParse(s string) (t time.Time, err error) {
	if t, err = time.Parse(TimeLayout, s); err == nil {
		return
	}
	return time.Parse(DateLayout, s)
}

// Index returns a map where key is entity, and value show if the entity
// should be processed (true) or not (false).
func (el *EntityList) Index() map[string]*EntityItem {
	el.mu.RLock()
	defer el.mu.RUnlock()
	return el.index
}

func (el *EntityList) Get(id string) (*EntityItem, bool) {
	if el == nil {
		return nil, false
	}
	el.mu.RLock()
	defer el.mu.RUnlock()
	item, ok := el.index[id]
	return item, ok
}

type EntityIndex map[string]bool

// IsExcluded returns true if the entity is excluded (is in the list, and has
// value false).
func (ei EntityIndex) IsExcluded(ent string) bool {
	v, ok := ei[ent]
	return ok && !v
}

// IsIncluded returns true if the entity is included (is in the list, and has
// value true).
func (ei EntityIndex) IsIncluded(ent string) bool {
	v, ok := ei[ent]
	return ok && v
}

// HasIncludes returns true if there's any included entities.
func (el *EntityList) HasIncludes() bool {
	el.mu.RLock()
	defer el.mu.RUnlock()
	return el.hasIncludes
}

// HasExcludes returns true if there's any excluded entities.
func (el *EntityList) HasExcludes() bool {
	el.mu.RLock()
	defer el.mu.RUnlock()
	return el.hasExcludes
}

// HasExplicitIncludes reports whether the input included a positive entry,
// even if every positive entry was overridden by an exclusion.
func (el *EntityList) HasExplicitIncludes() bool {
	el.mu.RLock()
	defer el.mu.RUnlock()
	return el.hasExplicitIncludes
}

// IsEmpty returns true if there's no entries in the list.
func (el *EntityList) IsEmpty() bool {
	return len(el.index) == 0
}

type entryIndex struct {
	index               map[string]bool
	hasExplicitIncludes bool
}

func buildEntryIndex(links []string) (entryIndex, error) {
	entries := entryIndex{index: make(map[string]bool, len(links))}
	var excluded []string
	var files []string
	// add all included items
	for _, ent := range links {
		if ent == "" {
			continue
		}
		parts := getTimeTuple(ent)
		switch {
		case HasExcludePrefix(parts[0]):
			trimmed := strings.TrimPrefix(parts[0], excludePrefix)
			if trimmed == "" {
				continue
			}
			sl, err := ParseLink(trimmed)
			if err != nil {
				return entryIndex{}, err
			}
			parts[0] = sl.String()
			excluded = append(excluded, strings.Join(parts, timeSeparator))
		case hasFilePrefix(parts[0]):
			trimmed := strings.TrimPrefix(parts[0], filePrefix)
			if trimmed == "" {
				continue
			}
			files = append(files, trimmed)
		default:
			// no prefix
			sl, err := ParseLink(parts[0])
			if err != nil {
				return entryIndex{}, err
			}
			parts[0] = sl.String()
			entries.index[strings.Join(parts, timeSeparator)] = true
			entries.hasExplicitIncludes = true
		}
	}
	// process files
	for _, file := range files {
		fileEntries, err := loadEntityIndex(file)
		if err != nil {
			return entryIndex{}, err
		}
		entries.hasExplicitIncludes = entries.hasExplicitIncludes || fileEntries.hasExplicitIncludes
		for ent, include := range fileEntries.index {
			if include {
				entries.index[ent] = true
			} else {
				excluded = append(excluded, ent)
			}
		}
	}
	for _, ent := range excluded {
		entries.index[ent] = false
	}
	return entries, nil
}

// C returns a channel where all included entries are streamed.
// The channel is closed when all entries have been sent, or when the context
// is cancelled.
func (el *EntityList) C(ctx context.Context) <-chan EntityItem {
	ch := make(chan EntityItem)
	var include []EntityItem
	for _, item := range el.Index() {
		if item.Include {
			include = append(include, *item)
		}
	}

	go func() {
		defer close(ch)
		for _, ent := range include {
			select {
			case <-ctx.Done():
				return
			case ch <- ent:
			}
		}
	}()
	return ch
}
