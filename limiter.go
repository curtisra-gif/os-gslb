// limiter.go
package main

import (
	"net"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type bucketEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

type RRLimiter struct {
	sync.Mutex
	// Change map to store bucketEntry instead of just the limiter
	buckets map[string]*bucketEntry
	rps     float64
	burst   int
}

func NewRRLimiter(rps float64, burst int) *RRLimiter {
	r := &RRLimiter{
		buckets: make(map[string]*bucketEntry),
		rps:     rps,
		burst:   burst,
	}
	// Start the cleanup goroutine immediately
	go r.cleanupLoop()
	return r
}

func (r *RRLimiter) Allow(ip net.IP, domain string) bool {
	key := generateKey(ip, domain) // Use logic previously discussed

	r.Lock()
	defer r.Unlock()

	b, exists := r.buckets[key]
	if !exists {
		b = &bucketEntry{
			limiter: rate.NewLimiter(rate.Limit(r.rps), r.burst),
		}
		r.buckets[key] = b
	}
	
	b.lastAccess = time.Now() // Refresh the TTL
	return b.limiter.Allow()
}

func (r *RRLimiter) cleanupLoop() {
	// Check every minute
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		r.Lock()
		now := time.Now()
		for key, b := range r.buckets {
			// If the bucket hasn't been used in 10 minutes, delete it
			if now.Sub(b.lastAccess) > 10*time.Minute {
				delete(r.buckets, key)
			}
		}
		r.Unlock()
	}
}

// Global RRL instance
var rrl *RRLimiter

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
