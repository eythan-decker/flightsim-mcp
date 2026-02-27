//go:build integration

package simconnect_test

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eytandecker/flightsim-mcp/internal/simconnect"
	"github.com/eytandecker/flightsim-mcp/internal/state"
)

func integrationConfig(t *testing.T) simconnect.Config {
	t.Helper()
	host := os.Getenv("SIMCONNECT_HOST")
	if host == "" {
		host = "192.168.0.44"
	}
	port := 4500
	if p := os.Getenv("SIMCONNECT_PORT"); p != "" {
		var err error
		port, err = strconv.Atoi(p)
		if err != nil {
			t.Fatalf("bad SIMCONNECT_PORT: %v", err)
		}
	}
	return simconnect.Config{
		Host:    host,
		Port:    port,
		Timeout: 5 * time.Second,
		AppName: "FlightSim-MCP-IntTest",
	}
}

func TestIntegrationConnect(t *testing.T) {
	cfg := integrationConfig(t)
	c := simconnect.NewClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := c.Connect(ctx)
	require.NoError(t, err)
	assert.Equal(t, simconnect.StateConnected, c.State())

	// Drain the OPEN ack from SimConnect
	h, _, err := c.ReadNext()
	require.NoError(t, err)
	t.Logf("received OPEN ack: type=0x%X version=0x%X size=%d", h.Type, h.Version, h.Size)
	assert.Equal(t, simconnect.RecvOpen, h.Type)

	err = c.Close()
	require.NoError(t, err)
	assert.Equal(t, simconnect.StateDisconnected, c.State())
}

func TestIntegrationRegisterAndReadPosition(t *testing.T) {
	cfg := integrationConfig(t)
	c := simconnect.NewClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := c.Connect(ctx)
	require.NoError(t, err)
	defer c.Close()

	// Drain OPEN ack
	_, _, err = c.ReadNext()
	require.NoError(t, err)

	// Register PLANE LATITUDE
	err = c.AddToDataDefinition(1, simconnect.PlaneLatitude)
	require.NoError(t, err)

	// Request data
	err = c.RequestData(1, 0, 100)
	require.NoError(t, err)

	// Read response — should be SimObjectData
	h, data, err := c.ReadNext()
	require.NoError(t, err)
	t.Logf("response: type=0x%X size=%d payload=%d bytes", h.Type, h.Size, len(data))
	assert.Equal(t, simconnect.RecvSimObjectData, h.Type)

	// Should have at least the 28-byte SimObjectData header + 8 bytes float64
	require.GreaterOrEqual(t, len(data), 36)
}

func TestIntegrationFullPollerFlow(t *testing.T) {
	cfg := integrationConfig(t)
	c := simconnect.NewClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := c.Connect(ctx)
	require.NoError(t, err)
	defer c.Close()

	// Drain OPEN ack
	_, _, err = c.ReadNext()
	require.NoError(t, err)

	mgr := state.NewManager(5 * time.Second)
	pollerCfg := simconnect.PollerConfig{PollInterval: 500 * time.Millisecond}
	p := simconnect.NewPoller(c, mgr, pollerCfg)

	err = p.RegisterSimVars()
	require.NoError(t, err)

	pollerCtx, pollerCancel := context.WithTimeout(ctx, 10*time.Second)
	defer pollerCancel()

	go func() { _ = p.Start(pollerCtx) }()

	// Wait for state manager to have valid position data
	require.Eventually(t, func() bool {
		pos, err := mgr.GetPosition()
		if err != nil {
			return false
		}
		// Latitude should be a reasonable value (not zero)
		return pos.Latitude != 0 && pos.Longitude != 0
	}, 8*time.Second, 200*time.Millisecond, "expected non-zero position data from sim")

	pos, err := mgr.GetPosition()
	require.NoError(t, err)
	t.Logf("position: lat=%.6f lon=%.6f alt=%.0f", pos.Latitude, pos.Longitude, pos.AltitudeMSL)

	// Sanity check: latitude and longitude should be in valid ranges
	assert.InDelta(t, 0, pos.Latitude, 90, "latitude should be in [-90, 90]")
	assert.InDelta(t, 0, pos.Longitude, 180, "longitude should be in [-180, 180]")
}
