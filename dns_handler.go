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
	remoteAddr, _, _ := net.SplitHostPort(w.RemoteAddr().String())
	clientIP := net.ParseIP(remoteAddr)

	// Initialize the logger with context for this request
	reqLogger := logger.With(
		zap.String("remote_addr", remoteAddr),
		zap.Uint16("msg_id", r.Id),
	)

	// --- EDNS Client Subnet (ECS) Extraction ---
	for _, extra := range r.Extra {
		if opt, ok := extra.(*dns.OPT); ok {
			for _, subOpt := range opt.Option {
				if ecs, ok := subOpt.(*dns.EDNS0_SUBNET); ok {
					clientIP = ecs.Address
					reqLogger.Debug("found ECS subnet", zap.String("ecs_ip", clientIP.String()))
				}
			}
		}
	}

	msg := new(dns.Msg)
	msg.SetReply(r)

	for _, q := range r.Question {
		// Response Rate Limiting (RRL) check
		if !rrl.Allow(clientIP, q.Name) {
			reqLogger.Warn("rate limit exceeded, slipping with TC bit", zap.String("qname", q.Name))
			msg.Truncated = true 
			w.WriteMsg(msg)
			return
		}

		zone, exists := h.Config.Zones[q.Name]
		if !exists || q.Qtype != dns.TypeA {
			reqLogger.Debug("unhandled query", zap.String("qname", q.Name), zap.Uint16("qtype", q.Qtype))
			continue
		}

		// 1. Find the closest pool based on GeoIP
		bestPool := h.findClosestPool(clientIP, zone.Pools)
		
		// 2. Try to get healthy IPs from the best pool
		answers := getHealthyIPs(bestPool, q.Name, zone.TTL)

		// 3. FAILOVER: If best pool is unhealthy, check other pools
		if len(answers) == 0 {
			reqLogger.Warn("primary pool unhealthy, attempting failover", zap.String("primary", bestPool.Name))
			
			for _, pool := range zone.Pools {
				if pool.Name == bestPool.Name {
					continue 
				}
				
				answers = getHealthyIPs(pool, q.Name, zone.TTL)
				if len(answers) > 0 {
					reqLogger.Info("failover successful", zap.String("failover_region", pool.Name))
					break 
				}
			}
		}

		// 4. ULTIMATE FALLBACK: Return all IPs from primary if everything is down
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

// getHealthyIPs extracts only the healthy IPs from a pool
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
	// Safely load the current active DB reader
	currentDB := dbPtr.Load()
	if currentDB == nil {
		return pools[0]
	}

	record, err := currentDB.City(clientIP)
	if err != nil {
		return pools[0] 
	}

	var best *ServerPool
	minDist := -1.0
	for _, p := range pools {
		dist := (record.Location.Latitude-p.Lat)*(record.Location.Latitude-p.Lat) +
			(record.Location.Longitude-p.Lon)*(record.Location.Longitude-p.Lon)
		if minDist == -1 || dist < minDist {
			minDist = dist
			best = p
		}
	}
	return best
}