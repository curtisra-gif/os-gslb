package config

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v3"
)

type Backend struct {
	IP     string `yaml:"ip"`
	Weight int    `yaml:"weight"`
}

type Region struct {
	Name     string    `yaml:"name"`
	DNSName  string    `yaml:"dns_name"`
	Backends []Backend `yaml:"backends"`
}

type Config struct {
	ListenIP               string   `yaml:"listen_ip"`
	ListenPort             int      `yaml:"listen_port"`
	RateLimit              int      `yaml:"rate_limit"`
	GeoDBPath              string   `yaml:"geo_db_path"`
	HealthCheckIntervalSec int      `yaml:"health_check_interval_sec"`
	Domain                 string   `yaml:"domain"`
	Regions                []Region `yaml:"regions"`
}

// LoadConfig loads YAML config from the specified path
func LoadConfig(path string) *Config {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("Failed to parse YAML config: %v", err)
	}

	return &cfg
}
