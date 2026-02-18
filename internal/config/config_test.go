package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadDefaults(t *testing.T) {
	cfg := Load()
	assert.Equal(t, "192.168.10.100", cfg.SimConnect.Host)
	assert.Equal(t, 4500, cfg.SimConnect.Port)
	assert.Equal(t, 10*time.Second, cfg.SimConnect.Timeout)
	assert.Equal(t, "flightsim-mcp", cfg.SimConnect.AppName)
	assert.Equal(t, 500*time.Millisecond, cfg.Polling.Interval)
	assert.Equal(t, 5*time.Second, cfg.Polling.StaleThreshold)
}

func TestLoadFromEnv(t *testing.T) {
	tests := []struct {
		name   string
		envKey string
		envVal string
		check  func(t *testing.T, cfg Config)
	}{
		{
			name:   "SIMCONNECT_HOST",
			envKey: "SIMCONNECT_HOST",
			envVal: "10.0.0.5",
			check: func(t *testing.T, cfg Config) {
				assert.Equal(t, "10.0.0.5", cfg.SimConnect.Host)
			},
		},
		{
			name:   "SIMCONNECT_PORT valid",
			envKey: "SIMCONNECT_PORT",
			envVal: "9999",
			check: func(t *testing.T, cfg Config) {
				assert.Equal(t, 9999, cfg.SimConnect.Port)
			},
		},
		{
			name:   "SIMCONNECT_PORT invalid falls back to default",
			envKey: "SIMCONNECT_PORT",
			envVal: "notanumber",
			check: func(t *testing.T, cfg Config) {
				assert.Equal(t, 4500, cfg.SimConnect.Port)
			},
		},
		{
			name:   "SIMCONNECT_TIMEOUT valid",
			envKey: "SIMCONNECT_TIMEOUT",
			envVal: "30s",
			check: func(t *testing.T, cfg Config) {
				assert.Equal(t, 30*time.Second, cfg.SimConnect.Timeout)
			},
		},
		{
			name:   "SIMCONNECT_TIMEOUT invalid falls back to default",
			envKey: "SIMCONNECT_TIMEOUT",
			envVal: "badvalue",
			check: func(t *testing.T, cfg Config) {
				assert.Equal(t, 10*time.Second, cfg.SimConnect.Timeout)
			},
		},
		{
			name:   "SIMCONNECT_APP_NAME",
			envKey: "SIMCONNECT_APP_NAME",
			envVal: "my-app",
			check: func(t *testing.T, cfg Config) {
				assert.Equal(t, "my-app", cfg.SimConnect.AppName)
			},
		},
		{
			name:   "POLL_INTERVAL valid",
			envKey: "POLL_INTERVAL",
			envVal: "250ms",
			check: func(t *testing.T, cfg Config) {
				assert.Equal(t, 250*time.Millisecond, cfg.Polling.Interval)
			},
		},
		{
			name:   "POLL_INTERVAL invalid falls back to default",
			envKey: "POLL_INTERVAL",
			envVal: "xyz",
			check: func(t *testing.T, cfg Config) {
				assert.Equal(t, 500*time.Millisecond, cfg.Polling.Interval)
			},
		},
		{
			name:   "STALE_THRESHOLD valid",
			envKey: "STALE_THRESHOLD",
			envVal: "10s",
			check: func(t *testing.T, cfg Config) {
				assert.Equal(t, 10*time.Second, cfg.Polling.StaleThreshold)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envKey, tt.envVal)
			cfg := Load()
			tt.check(t, cfg)
		})
	}
}
