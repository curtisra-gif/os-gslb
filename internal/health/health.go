package health

import (
	"time"

	"gslb/internal/model"
)

type Manager struct {}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) Start(record *model.Record) {
	if record.HealthCheck == nil {
		return
	}
	go func() {
		d, _ := time.ParseDuration(record.HealthCheck.Frequency)
		for range time.Tick(d) {
			for i := range record.IPs {
				record.IPs[i].Alive = true
			}
		}
	}()
}
