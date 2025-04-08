package client

// strategy is an interface that defines the strategy for selecting the next
// item.
type strategy interface {
	// next returns the next item in the pool.
	next() int
}

// roundRobin implements the round-robin strategy.
type roundRobin struct {
	// total is the total number of items in the pool.
	total int
	// i is the current item index.
	i int
}

// newRoundRobin creates a new round-robin strategy with the given total number
// of items.
func newRoundRobin(total int) *roundRobin {
	return &roundRobin{total: total}
}

func (r *roundRobin) next() int {
	r.i = (r.i + 1) % r.total
	return r.i
}
