package main

import (
	"sync"

	"golang.org/x/time/rate"
)

var (
	limiters = make(map[string]*rate.Limiter)
	limitMu  sync.Mutex
)

// getLimiter returns a rate limiter for a specific client IP
func getLimiter(ip string) *rate.Limiter {
	limitMu.Lock()
	defer limitMu.Unlock()

	if l, v := limiters[ip]; v {
		return l
	}

	// You can pull these values from cfg.Server.Throttling if you pass cfg here
	l := rate.NewLimiter(rate.Limit(10), 20)
	limiters[ip] = l
	return l
}
