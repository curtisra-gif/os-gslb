package health

import (
	"net"
	"sync"
	"time"
)

type HealthChecker struct {
	backends map[string]bool
	mu       sync.RWMutex
	interval time.Duration
}

func New(intervalSec int) *HealthChecker {
	return &HealthChecker{
		backends: make(map[string]bool),
		interval: time.Duration(intervalSec) * time.Second,
	}
}

func (h *HealthChecker) Start(backends []string) {
	go func() {
		for {
			for _, ip := range backends {
				h.mu.Lock()
				h.backends[ip] = h.check(ip)
				h.mu.Unlock()
			}
			time.Sleep(h.interval)
		}
	}()
}

func (h *HealthChecker) check(ip string) bool {
	conn, err := net.DialTimeout("tcp", ip+":53", 2*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func (h *HealthChecker) IsHealthy(ip string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.backends[ip]
}
