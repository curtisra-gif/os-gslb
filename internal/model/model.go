package model

type IP struct {
	Address string `json:"address"`
	Region  string `json:"region,omitempty"`
	Weight  int    `json:"weight,omitempty"`
	Alive   bool
}

type HealthCheck struct {
	Type           string `json:"type"`
	Port           int    `json:"port,omitempty"`
	Path           string `json:"path,omitempty"`
	HTTPS          bool   `json:"https,omitempty"`
	ExpectedStatus int    `json:"http_status_code,omitempty"`
	Frequency      string `json:"frequency"`
}

type Record struct {
	Name        string        `json:"name"`
	IPs         []IP          `json:"ips"`
	TTL         uint32        `json:"ttl"`
	HealthCheck *HealthCheck `json:"health_check,omitempty"`
}
