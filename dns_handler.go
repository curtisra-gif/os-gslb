package main

import (
	"net"

	"github.com/miekg/dns"
)

type GSLBHandler struct {
	Config *Config
}

func (h *GSLBHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	remoteAddr, _, _ := net.SplitHostPort(w.RemoteAddr().String())
	clientIP := net.ParseIP(remoteAddr)

	// --- EDNS Client Subnet (ECS) Extraction ---
	for _, extra := range r.Extra {
		if opt, ok := extra.(*dns.OPT); ok {
			for _, subOpt := range opt.Option {
				if ecs, ok := subOpt.(*dns.EDNS0_SUBNET); ok {
					clientIP = ecs.Address
				}
			}
		}
	}

	// Throttling Check (simplified for example)
	if !getLimiter(remoteAddr).Allow() {
		return
	}

	msg := new(dns.Msg)
	msg.SetReply(r)

	for _, q := range r.Question {
		zone, exists := h.Config.Zones[q.Name]
		if !exists || q.Qtype != dns.TypeA {
			continue
		}

		bestPool := h.findClosestPool(clientIP, zone.Pools)

		for _, ip := range bestPool.IPs {
			if bestPool.Healthy[ip] {
				msg.Answer = append(msg.Answer, &dns.A{
					Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: zone.TTL},
					A:   net.ParseIP(ip),
				})
			}
		}
	}
	w.WriteMsg(msg)
}

func (h *GSLBHandler) findClosestPool(clientIP net.IP, pools []*ServerPool) *ServerPool {
	record, err := db.City(clientIP)
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
