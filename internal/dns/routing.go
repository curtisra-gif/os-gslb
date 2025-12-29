package dns

import (
	"math/rand"

	"github.com/curtisra-gif/os-gslb/internal/config"
	"github.com/curtisra-gif/os-gslb/internal/health"
)

type Router struct {
	cfg    *config.Config
	health *health.HealthChecker
}

func NewRouter(cfg *config.Config, health *health.HealthChecker) *Router {
	return &Router{cfg: cfg, health: health}
}

// Selects a backend IP based on region and weight
func (r *Router) Route(regionName string) string {
	for _, region := range r.cfg.Regions {
		if region.Name == regionName {
			totalWeight := 0
			for _, b := range region.Backends {
				if r.health.IsHealthy(b.IP) {
					totalWeight += b.Weight
				}
			}

			randVal := rand.Intn(totalWeight)
			for _, b := range region.Backends {
				if !r.health.IsHealthy(b.IP) {
					continue
				}
				if randVal < b.Weight {
					return b.IP
				}
				randVal -= b.Weight
			}
		} else {

		}
	}
	return "" // fallback if no healthy backend
}
