package structures

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
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
	index       map[string]*EntityItem
	mu          sync.RWMutex
	hasIncludes bool
	hasExcludes bool
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

// NewEntityList creates an EntityList from a slice of IDs or URLs (entites).
func NewEntityList(entities []string) (*EntityList, error) {
	var el EntityList

	index, err := buildEntryIndex(entities)
	if err != nil {
		return nil, err
	}
	el.fromIndex(index)

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

func NewEntityListFromItems(items ...EntityItem) *EntityList {
	el := EntityList{
		index: make(map[string]*EntityItem, len(items)),
	}
	for _, item := range items {
		el.index[item.Id] = &item
		if item.Include && !el.hasIncludes {
			el.hasIncludes = true
		} else if !item.Include && !el.hasExcludes {
			el.hasExcludes = true
		}
	}
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

// LoadEntityList creates an EntityList from a slice of IDs or URLs (entites).
func loadEntityIndex(filename string) (map[string]bool, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readEntityIndex(f, maxFileEntries)
}

// readEntityList is a rather naïve implementation that reads the entire file up
// to maxEntries entities (empty lines are skipped), and populates the slice of
// strings, which is then passed to NewEntityList.  On large lists it will
// probably use a silly amount of memory.
func readEntityIndex(r io.Reader, maxEntries int) (map[string]bool, error) {
	br := bufio.NewReader(r)
	var elements []string
	total := 0
	var exit bool
	for n := 1; ; n++ {
		if total >= maxEntries {
			return nil, fmt.Errorf("%w (%d)", ErrMaxFileSize, maxFileEntries)
		}
		line, err := br.ReadString('\n')
		if errors.Is(err, io.EOF) {
			exit = true
		} else if err != nil {
			return nil, err
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
		if include {
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
	return el.hasIncludes
}

// HasExcludes returns true if there's any excluded entities.
func (el *EntityList) HasExcludes() bool {
	return el.hasExcludes
}

// IsEmpty returns true if there's no entries in the list.
func (el *EntityList) IsEmpty() bool {
	return len(el.index) == 0
}

func buildEntryIndex(links []string) (map[string]bool, error) {
	index := make(map[string]bool, len(links))
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
				return nil, err
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
				return nil, err
			}
			parts[0] = sl.String()
			index[strings.Join(parts, timeSeparator)] = true
		}
	}
	// process files
	for _, file := range files {
		index2, err := loadEntityIndex(file)
		if err != nil {
			return nil, err
		}
		for ent, include := range index2 {
			if include {
				index[ent] = true
			} else {
				excluded = append(excluded, ent)
			}
		}
	}
	for _, ent := range excluded {
		index[ent] = false
	}
	return index, nil
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
