package main

import (
	"net"
	"time"
)

func RunHealthChecks(cfg *Config) {
	ticker := time.NewTicker(15 * time.Second)
	for range ticker.C {
		for _, zone := range cfg.Zones {
			for _, pool := range zone.Pools {
				for _, ip := range pool.IPs {
					// Check port 80 as a default; could be moved to config
					conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, "80"), 2*time.Second)
					pool.Healthy[ip] = (err == nil)
					if err == nil {
						conn.Close()
					}
				}
			}
		}
	}
}
