package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		ListenAddr string `yaml:"listen_addr"`
		GeoDBPath  string `yaml:"geoip_db_path"`
		Throttling struct {
			RPS   float64 `yaml:"rps"`
			Burst int     `yaml:"burst"`
		} `yaml:"throttling"`
	} `yaml:"server"`
	Zones map[string]ZoneConfig `yaml:"zones"`
}

type ZoneConfig struct {
	TTL   uint32        `yaml:"ttl"`
	Pools []*ServerPool `yaml:"pools"`
}

type ServerPool struct {
	Name    string   `yaml:"name"`
	Lat     float64  `yaml:"lat"`
	Lon     float64  `yaml:"lon"`
	IPs     []string `yaml:"ips"`
	Healthy map[string]bool
}

func LoadConfig(path string) (*Config, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(buf, &cfg); err != nil {
		return nil, err
	}
	// Initialize health maps
	for _, zone := range cfg.Zones {
		for _, pool := range zone.Pools {
			pool.Healthy = make(map[string]bool)
		}
	}
	return &cfg, nil
}
