package primitive

import "sync"

// Counter is the thread safe Counter.
type Counter struct {
	n  int
	mu sync.Mutex // guards refcnt
}

func (c *Counter) Add(n int) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.n += n
	return c.n
}

func (c *Counter) Inc() int {
	return c.Add(1)
}

func (c *Counter) Dec() int {
	return c.Add(-1)
}

func (c *Counter) N() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}
