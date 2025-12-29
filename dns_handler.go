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

	// Create a logger for this specific request context
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
		// Response Rate Limiting (RRL) check per domain/subnet
		if !rrl.Allow(clientIP, q.Name) {
			reqLogger.Warn("rate limit exceeded, slipping with TC bit", zap.String("qname", q.Name))
			msg.Truncated = true // Force legitimate clients to retry via TCP
			w.WriteMsg(msg)
			return
		}

		zone, exists := h.Config.Zones[q.Name]
		if !exists || q.Qtype != dns.TypeA {
			reqLogger.Debug("unhandled query", zap.String("qname", q.Name), zap.Uint16("qtype", q.Qtype))
			continue
		}

		bestPool := h.findClosestPool(clientIP, zone.Pools)
		reqLogger.Info("routing query", zap.String("qname", q.Name), zap.String("selected_pool", bestPool.Name))

		bestPool.Mu.RLock()
		var answers []dns.RR
		for _, ip := range bestPool.IPs {
			if bestPool.Healthy[ip] {
				answers = append(answers, &dns.A{
					Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: zone.TTL},
					A:   net.ParseIP(ip),
				})
			}
		}
		bestPool.Mu.RUnlock()

		// Fallback: If no healthy servers, return all IPs as a last resort
		if len(answers) == 0 {
			reqLogger.Error("no healthy IPs in pool, using fallback", zap.String("pool", bestPool.Name))
			for _, ip := range bestPool.IPs {
				answers = append(answers, &dns.A{
					Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: zone.TTL},
					A:   net.ParseIP(ip),
				})
			}
		}
		msg.Answer = append(msg.Answer, answers...)
	}

	w.WriteMsg(msg)
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