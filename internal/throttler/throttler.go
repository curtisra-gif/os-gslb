package throttler

import (
	"sync"
	"time"
)

type Throttler struct {
	limit int
	mu    sync.Mutex
	data  map[string]*client
}

type client struct {
	last  time.Time
	count int
}

func New(limit int) *Throttler {
	return &Throttler{
		limit: limit,
		data:  make(map[string]*client),
	}
}

func (t *Throttler) Allow(ip string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	c, ok := t.data[ip]
	now := time.Now()

	if !ok || now.Sub(c.last) > time.Second {
		t.data[ip] = &client{last: now, count: 1}
		return true
	}

	if c.count < t.limit {
		c.count++
		return true
	}

	return false
}
