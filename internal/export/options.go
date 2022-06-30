package export

import (
	"fmt"
	"time"

	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/logger"
)

// Options allows to configure slack export options.
type Options struct {
	Oldest       time.Time
	Latest       time.Time
	IncludeFiles bool
	Logger       logger.Interface
	Include      []string
	Exclude      []string
}

type filterFunc func(s string) bool

// Filter returns a makeFilterFn function that returns true if the channel is eligible
// for export or false - if not.  It builds an internal map of channels based on
// Include and Exclude slices.
func (opts Options) makeFilterFn() (filterFunc, error) {
	if len(opts.Include) == 0 && len(opts.Exclude) == 0 {
		// no filters applied
		return func(string) bool { return true }, nil
	}
	m, err := opts.mkChanMap()
	if err != nil {
		return nil, err
	}
	filterFn := func(ch string) bool {
		shouldExport, ok := m[ch]
		if !ok {
			return true // export by default
		}
		return shouldExport
	}
	return filterFn, nil
}

func (opts Options) mkChanMap() (map[string]bool, error) {
	var m = make(map[string]bool, len(opts.Include)+len(opts.Exclude))
	if err := normaliseChans(opts.Include); err != nil {
		return nil, fmt.Errorf("error normalising include list: %w", err)
	}
	if err := normaliseChans(opts.Exclude); err != nil {
		return nil, fmt.Errorf("error normalising exclude list: %w", err)
	}
	for _, id := range opts.Include {
		m[id] = true
	}
	for _, id := range opts.Exclude {
		m[id] = false
	}
	return m, nil
}

// normaliseChans normalises all channels to ID form.  If the idsOrURLs[i] is
// a channel ID, it is unmodified, if it is URL - it is parsed and replaced
// with the channel ID.
func normaliseChans(idsOrURLs []string) error {
	for i, val := range idsOrURLs {
		if val == "" {
			continue
		}
		if !structures.IsURL(val) {
			continue
		}
		ch, err := structures.ParseURL(val)
		if err != nil {
			return fmt.Errorf("not a valid Slack URL %q: %w", val, err)
		}
		if !ch.IsValid() {
			return fmt.Errorf("not a valid slack URL: %s", val)
		}
		idsOrURLs[i] = ch.Channel
	}
	return nil
}
