package edge

import (
	"time"

	"golang.org/x/time/rate"
)

type tier struct {
	// once eveyr
	t time.Duration
	// burst
	b int
}

func (t tier) limiter() *rate.Limiter {
	return rate.NewLimiter(rate.Every(t.t), t.b)
}

var (
	// tier1 = tier{t: 1 * time.Minute, b: 2}
	// tier2 = tier{t: 3 * time.Second, b: 3}
	tier2boost = tier{t: 300 * time.Millisecond, b: 5}
	tier3      = tier{t: 1200 * time.Millisecond, b: 4}
	// tier4 = tier{t: 60 * time.Millisecond, b: 5}
)
