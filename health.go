package main

import (
	"fmt"
	"log"
	"net"
	"time"
)

func RunHealthChecks(cfg *Config) {
	// Initial check
	performCheck(cfg)

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		performCheck(cfg)
	}
}

func performCheck(cfg *Config) {
	for zoneName, zone := range cfg.Zones {
		for _, pool := range zone.Pools {
			for _, ip := range pool.IPs {
				addr := fmt.Sprintf("%s:%d", ip, pool.MonitorPort)

				// Default to TCP if not specified
				proto := pool.MonitorProto
				if proto == "" {
					proto = "tcp"
				}

				conn, err := net.DialTimeout(proto, addr, 2*time.Second)

				pool.Mu.Lock()
				isHealthy := (err == nil)
				pool.Healthy[ip] = isHealthy
				pool.Mu.Unlock()

				if err != nil {
					log.Printf("[HEALTH] %s (%s) DOWN: %v", zoneName, ip, err)
				} else {
					conn.Close()
				}
			}
		}
	}
}
