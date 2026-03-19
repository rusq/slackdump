package stream

import (
	"sync"

	"github.com/rusq/slack"
)

// chanCache is used to cache channel info to avoid fetching it multiple times.
type chanCache struct {
	m sync.Map
}

// get returns the channel info from the cache.  If it fails to find it, it
// returns nil.
func (c *chanCache) get(key string) *slack.Channel {
	v, ok := c.m.Load(key)
	if !ok {
		return nil
	}
	return v.(*slack.Channel)
}

// set sets the channel info in the cache under the respective key.
func (c *chanCache) set(key string, ch *slack.Channel) {
	c.m.Store(key, ch)
}

// userCache is used to cache channel user IDs to avoid fetching them multiple times.
type userCache struct {
	m sync.Map
}

// get returns the user IDs from the cache.  If it fails to find them, it
// returns nil.
func (c *userCache) get(key string) []string {
	v, ok := c.m.Load(key)
	if !ok {
		return nil
	}
	return v.([]string)
}

// set sets the user IDs in the cache under the channel ID key.
func (c *userCache) set(key string, users []string) {
	c.m.Store(key, users)
}
