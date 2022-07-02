package structures

import (
	"bufio"
	"io"
	"os"
	"sort"
	"strings"

	"errors"
)

const (
	// excludePrefix is the prefix that is used to mark channel exclusions, i.e.
	// for export or when downloading conversations.
	excludePrefix = "^"
	filePrefix    = "@"

	// maxFileEntries is the maximum non-empty entries that will be read from
	// the file. Who ever needs more than 64Ki channels.
	maxFileEntries = 65536
)

// EntityList is an Inclusion/Exclusion list
type EntityList struct {
	Include []string
	Exclude []string
}

func HasExcludePrefix(s string) bool {
	return strings.HasPrefix(s, excludePrefix)
}

func hasFilePrefix(s string) bool {
	return strings.HasPrefix(s, filePrefix)
}

// MakeEntityList creates an EntityList from a slice of IDs or URLs (entites).
func MakeEntityList(entities []string) (*EntityList, error) {
	var el EntityList

	index, err := buildEntityIndex(entities)
	if err != nil {
		return nil, err
	}
	el.fromIndex(index)

	return &el, nil
}

// MakeEntityList creates an EntityList from a slice of IDs or URLs (entites).
func LoadEntityList(filename string) (*EntityList, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readEntityList(f, maxFileEntries)
}

// readEntityList is a rather naïve implementation that reads the entire file up
// to maxEntries entities (empty lines are skipped), and populates the slice of
// strings, which is then passed to NewEntityList.  On large lists it will
// probably use a silly amount of memory.
func readEntityList(r io.Reader, maxEntries int) (*EntityList, error) {
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
	return MakeEntityList(elements)
}

func (el *EntityList) fromIndex(index map[string]bool) {
	for ent, include := range index {
		if include {
			el.Include = append(el.Include, ent)
		} else {
			el.Exclude = append(el.Exclude, ent)
		}
	}
	sort.Strings(el.Include)
	sort.Strings(el.Exclude)
}

// Index returns a map where key is entity, and value show if the entity
// should be processed (true) or not (false).
func (el *EntityList) Index() map[string]bool {
	var idx = make(map[string]bool, len(el.Include)+len(el.Exclude))
	for _, v := range el.Include {
		idx[v] = true
	}
	for _, v := range el.Exclude {
		idx[v] = false
	}
	return idx
}

func (el *EntityList) HasIncludes() bool {
	return len(el.Include) > 0
}

func (el *EntityList) HasExcludes() bool {
	return len(el.Exclude) > 0
}

func (el *EntityList) IsEmpty() bool {
	return len(el.Include)+len(el.Exclude) == 0
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
		switch {
		case HasExcludePrefix(ent):
			trimmed := strings.TrimPrefix(ent, excludePrefix)
			if trimmed == "" {
				continue
			}
			sl, err := ParseLink(trimmed)
			if err != nil {
				return nil, err
			}
			excluded = append(excluded, sl.String())
		case hasFilePrefix(ent):
			trimmed := strings.TrimPrefix(ent, filePrefix)
			if trimmed == "" {
				continue
			}
			files = append(files, trimmed)
		default:
			sl, err := ParseLink(ent)
			if err != nil {
				return nil, err
			}
			index[sl.String()] = true
		}
	}
	// process files
	for _, file := range files {
		el, err := LoadEntityList(file)
		if err != nil {
			return nil, err
		}
		for ent, include := range el.Index() {
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
