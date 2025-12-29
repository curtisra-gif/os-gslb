package config

import (
	"encoding/json"
	"os"

	"gslb/internal/model"
)

type Config struct {
	Records []model.Record `json:"records"`
}

func Load(path string) *Config {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		panic(err)
	}
	return &c
}
