package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration.
type Config struct {
	SimConnect SimConnectConfig
	Polling    PollingConfig
}

// SimConnectConfig holds SimConnect TCP connection settings.
type SimConnectConfig struct {
	Host    string
	Port    int
	Timeout time.Duration
	AppName string
}

// PollingConfig holds data polling settings.
type PollingConfig struct {
	Interval       time.Duration
	StaleThreshold time.Duration
}

// Load reads configuration from environment variables, falling back to defaults.
func Load() Config {
	return Config{
		SimConnect: SimConnectConfig{
			Host:    getEnvString("SIMCONNECT_HOST", "192.168.10.100"),
			Port:    getEnvInt("SIMCONNECT_PORT", 4500),
			Timeout: getEnvDuration("SIMCONNECT_TIMEOUT", 10*time.Second),
			AppName: getEnvString("SIMCONNECT_APP_NAME", "flightsim-mcp"),
		},
		Polling: PollingConfig{
			Interval:       getEnvDuration("POLL_INTERVAL", 500*time.Millisecond),
			StaleThreshold: getEnvDuration("STALE_THRESHOLD", 5*time.Second),
		},
	}
}

func getEnvString(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return defaultVal
	}
	return d
}
