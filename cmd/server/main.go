package main

import (
	"flag"
	"log"
	"net"

	"github.com/curtisra-gif/os-gslb/internal/config"
	gdns "github.com/curtisra-gif/os-gslb/internal/dns"
	"github.com/curtisra-gif/os-gslb/internal/health"
	"github.com/curtisra-gif/os-gslb/internal/throttler"
	mdns "github.com/miekg/dns"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to YAML config file")
	flag.Parse()

	cfg := config.LoadConfig(*configPath)

	// Initialize throttler
	throttler := throttler.New(cfg.RateLimit)

	// Gather all backend IPs for health checking
	var allBackends []string
	for _, region := range cfg.Regions {
		for _, b := range region.Backends {
			allBackends = append(allBackends, b.IP)
		}
	}

	// Health checker
	healthChecker := health.New(cfg.HealthCheckIntervalSec)
	healthChecker.Start(allBackends)

	// Geo locator
	geoLocator, err := gdns.NewGeoLocator(cfg.GeoDBPath)
	if err != nil {
		log.Fatalf("Failed to load GeoIP2 DB: %v", err)
	}

	// Router
	router := gdns.NewRouter(cfg, healthChecker)

	// DNS handler
	handler := &gdns.Handler{
		Cfg:        cfg,
		Throttler:  throttler,
		GeoLocator: geoLocator,
		Router:     router,
	}

	addr := net.UDPAddr{
		IP:   net.ParseIP(cfg.ListenIP),
		Port: cfg.ListenPort,
	}

	log.Printf("Starting GSLB DNS server for domain %s on %s:%d", cfg.Domain, cfg.ListenIP, cfg.ListenPort)
	server := &mdns.Server{
		Addr:    addr.String(),
		Net:     "udp",
		Handler: handler,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start DNS server: %v", err)
	}
}
