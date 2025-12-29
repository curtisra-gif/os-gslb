package main

import (
	"net"

	"github.com/miekg/dns"
	"go.uber.org/zap"
)

type GSLBHandler struct {
	Config *Config
}

func (h *GSLBHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
    // ... (previous logic for remoteAddr and RRL remains the same) ...

    for _, q := range r.Question {
        zone, exists := h.Config.Zones[q.Name]
        if !exists || q.Qtype != dns.TypeA {
            continue
        }

        // 1. Find the closest pool based on GeoIP
        bestPool := h.findClosestPool(clientIP, zone.Pools)
        
        // 2. Try to get healthy IPs from the best pool
        answers := getHealthyIPs(bestPool, q.Name, zone.TTL)

        // 3. FAILOVER: If best pool is empty/unhealthy, check other pools
        if len(answers) == 0 {
            reqLogger.Warn("primary pool unhealthy, attempting failover", 
                zap.String("primary", bestPool.Name))
            
            for _, pool := range zone.Pools {
                if pool.Name == bestPool.Name {
                    continue // Skip the one we already checked
                }
                
                answers = getHealthyIPs(pool, q.Name, zone.TTL)
                if len(answers) > 0 {
                    reqLogger.Info("failover successful", 
                        zap.String("failover_region", pool.Name))
                    break // Stop at the first healthy alternative region
                }
            }
        }

        // 4. ULTIMATE FALLBACK: If EVERYTHING is down, return primary pool IPs anyway
        if len(answers) == 0 {
            reqLogger.Error("all regions unhealthy, returning primary pool as last resort")
            bestPool.Mu.RLock()
            for _, ip := range bestPool.IPs {
                answers = append(answers, &dns.A{
                    Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: zone.TTL},
                    A:   net.ParseIP(ip),
                })
            }
            bestPool.Mu.RUnlock()
        }
        
        msg.Answer = append(msg.Answer, answers...)
    }
    w.WriteMsg(msg)
}

// Helper function to extract healthy IPs from a pool
func getHealthyIPs(pool *ServerPool, qname string, ttl uint32) []dns.RR {
    var rrs []dns.RR
    pool.Mu.RLock()
    defer pool.Mu.RUnlock()
    
    for _, ip := range pool.IPs {
        if pool.Healthy[ip] {
            rrs = append(rrs, &dns.A{
                Hdr: dns.RR_Header{Name: qname, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: ttl},
                A:   net.ParseIP(ip),
            })
        }
    }
    return rrs
}

func (h *GSLBHandler) findClosestPool(clientIP net.IP, pools []*ServerPool) *ServerPool {
	record, err := db.City(clientIP)
	if err != nil {
		return pools[0] // Fallback to first pool if GeoIP lookup fails
	}

	var best *ServerPool
	minDist := -1.0
	for _, p := range pools {
		// Basic Euclidean distance calculation
		dist := (record.Location.Latitude-p.Lat)*(record.Location.Latitude-p.Lat) +
			(record.Location.Longitude-p.Lon)*(record.Location.Longitude-p.Lon)
		if minDist == -1 || dist < minDist {
			minDist = dist
			best = p
		}
	}
	return best
}