package lb

import (
	"math/rand"

	"gslb/internal/geo"
	"gslb/internal/model"
)

func Select(ips []model.IP, clientIP string) string {
	var healthy []model.IP
	for _, ip := range ips {
		if ip.Alive {
			healthy = append(healthy, ip)
		}
	}
	if len(healthy) == 0 {
		return ""
	}

	region := geo.Region(clientIP)

	var preferred []model.IP
	if region != "" {
		for _, ip := range healthy {
			if ip.Region == region {
				preferred = append(preferred, ip)
			}
		}
		if len(preferred) > 0 {
			return weighted(preferred)
		}
	}

	return weighted(healthy)
}

func weighted(ips []model.IP) string {
	total := 0
	for _, ip := range ips {
		w := ip.Weight
		if w <= 0 {
			w = 1
		}
		total += w
	}

	r := rand.Intn(total)
	for _, ip := range ips {
		w := ip.Weight
		if w <= 0 {
			w = 1
		}
		if r < w {
			return ip.Address
		}
		r -= w
	}
	return ips[0].Address
}
