package structures

import (
	"bufio"
	"io"
	"os"
	"strings"
	"time"

	"errors"
)

const (
	// excludePrefix is the prefix that is used to mark channel exclusions, i.e.
	// for export or when downloading conversations.
	excludePrefix = "^"
	filePrefix    = "@"
	timeSeparator = "|"
	timeFmt       = "2006-01-02T15:04:05"

	// maxFileEntries is the maximum non-empty entries that will be read from
	// the file. Who ever needs more than 64Ki channels.
	maxFileEntries = 65536
)

type EntityItem struct {
	Id      string
	Oldest  time.Time
	Latest  time.Time
	Include bool
}

// EntityList is an Inclusion/Exclusion list
type EntityList struct {
	index       map[string]*EntityItem
	hasIncludes bool
	hasExcludes bool
}

func HasExcludePrefix(s string) bool {
	return strings.HasPrefix(s, excludePrefix)
}

func hasFilePrefix(s string) bool {
	return strings.HasPrefix(s, filePrefix)
}

// NewEntityList creates an EntityList from a slice of IDs or URLs (entities).
func NewEntityList(entities []string) (*EntityList, error) {
	var el EntityList

	index, err := buildEntityIndex(entities)
	if err != nil {
		return nil, err
	}
	el.fromIndex(index)

	return &el, nil
}

// MakeEntityList creates an EntityList from a slice of IDs or URLs (entities).
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
	var total = 0
	var exit bool
	for n := 1; ; n++ {
		if total >= maxEntries {
			return nil, errors.New("maximum file size exceeded")
		}
		line, err := br.ReadString('\n')
		if err == io.EOF {
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
	return buildEntityIndex(elements)
}

func getTimeTuple(item string) []string {
	if strings.HasPrefix(item, filePrefix) {
		return []string{item}
	}
	return strings.SplitN(item, timeSeparator, 3)
}

func (el *EntityList) fromIndex(index map[string]bool) {
	el.index = make(map[string]*EntityItem, len(index))
	for ent, include := range index {
	  parts := getTimeTuple(ent)

	  item := &EntityItem{
	    Id: parts[0],
	    Include: include,
	  }
	  if len(parts) > 1 {
	    item.Oldest, _ = time.Parse(timeFmt, parts[1])
	  }
	  if len(parts) == 3 {
	    item.Latest, _ = time.Parse(timeFmt, parts[2])
	  }

	  el.index[item.Id] = item
	  if include {
			el.hasIncludes = true
	  } else {
			el.hasExcludes = true
	  }
	}
}

// Index returns a map where key is entity, and value show if the entity
// should be processed (true) or not (false).
func (el *EntityList) Index() map[string]*EntityItem {
	return el.index
}

func (el *EntityList) HasIncludes() bool {
	return el.hasIncludes
}

func (el *EntityList) HasExcludes() bool {
	return el.hasExcludes
}

func (el *EntityList) IsEmpty() bool {
	return len(el.index) == 0
}

func buildEntityIndex(entities []string) (map[string]bool, error) {
	var index = make(map[string]bool, len(entities))
	var excluded []string
	var files []string
	// add all included items
	for _, ent := range entities {
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
