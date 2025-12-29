package main

import (
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"
)

func RunHealthChecks(cfg *Config) {
	// Initial check on startup
	performCheck(cfg)

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		performCheck(cfg)
	}
}

func performCheck(cfg *Config) {
	for _, zone := range cfg.Zones {
		for _, pool := range zone.Pools {
			for _, ip := range pool.IPs {
				addr := fmt.Sprintf("%s:%d", ip, pool.MonitorPort)
				proto := pool.MonitorProto
				if proto == "" {
					proto = "tcp"
				}

				// Perform health check
				conn, err := net.DialTimeout(proto, addr, 3*time.Second)
				isHealthy := (err == nil)
				if conn != nil {
					conn.Close()
				}

				pool.Mu.Lock()
				// Track status changes to avoid noisy logs
				wasHealthy, exists := pool.Healthy[ip]
				pool.Healthy[ip] = isHealthy
				pool.Mu.Unlock()

				// Only log when the state changes
				if !exists || wasHealthy != isHealthy {
					if isHealthy {
						logger.Info("server is UP", 
							zap.String("ip", ip), 
							zap.String("pool", pool.Name),
						)
					} else {
						logger.Warn("server is DOWN", 
							zap.String("ip", ip), 
							zap.String("pool", pool.Name), 
							zap.Error(err),
						)
					}
				}
			}
		}
	}
}