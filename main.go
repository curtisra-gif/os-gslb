package main

import (
	"log"

	"github.com/miekg/dns"
	"github.com/oschwald/geoip2-golang"
	"go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

var db *geoip2.Reader

var logger *zap.Logger

func initLogger() {
    config := zap.NewProductionConfig()
    config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
    
    var err error
    logger, err = config.Build()
    if err != nil {
        panic(err)
    }
}

func watchGeoDB(path string) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		logger.Info("checking for GeoIP database updates")
		
		newDB, err := geoip2.Open(path)
		if err != nil {
			logger.Error("failed to open new GeoIP DB", zap.Error(err))
			continue
		}

		// Swap the old pointer with the new one
		oldDB := dbPtr.Swap(newDB)
		
		// Give existing queries a moment to finish before closing the old file
		time.Sleep(5 * time.Second)
		if oldDB != nil {
			oldDB.Close()
		}
		
		logger.Info("GeoIP database hot-reloaded successfully")
	}
}

func main() {
	initLogger()
    defer logger.Sync() // Flushes buffer before exit

	cfg, err := LoadConfig("config.yaml")
	if err != nil {
		logger.Fatal("config error", zap.Error(err))
	}

	// Initial Load
	initialDB, err := geoip2.Open(cfg.Server.GeoDBPath)
	if err != nil {
		logger.Fatal("initial geoip load failed", zap.Error(err))
	}
	dbPtr.Store(initialDB)

	// Start the daily update checker
	go watchGeoDB(cfg.Server.GeoDBPath)

	go RunHealthChecks(cfg)

	// Initialize RRL using config values
	rrl = NewRRLimiter(cfg.Server.Throttling.RPS, cfg.Server.Throttling.Burst)

	handler := &GSLBHandler{Config: cfg}
	server := &dns.Server{
		Addr:    cfg.Server.ListenAddr,
		Net:     "udp",
		Handler: handler,
	}

	log.Printf("GSLB active on %s", cfg.Server.ListenAddr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
