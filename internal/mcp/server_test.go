package mcp_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internalmcp "github.com/eytandecker/flightsim-mcp/internal/mcp"
	"github.com/eytandecker/flightsim-mcp/internal/simconnect"
	"github.com/eytandecker/flightsim-mcp/internal/state"
	"github.com/eytandecker/flightsim-mcp/pkg/types"
)

// mockPositionGetter controls what GetPosition returns in tests.
type mockPositionGetter struct {
	pos types.AircraftPosition
	err error
}

func (m *mockPositionGetter) GetPosition() (types.AircraftPosition, error) {
	return m.pos, m.err
}

var samplePos = types.AircraftPosition{
	Latitude:       47.6062,
	Longitude:      -122.3321,
	AltitudeMSL:    35000.0,
	AltitudeAGL:    34950.0,
	HeadingTrue:    270.0,
	HeadingMag:     268.5,
	IndicatedSpeed: 450.0,
	TrueSpeed:      455.0,
	GroundSpeed:    448.0,
	VerticalSpeed:  500.0,
	Pitch:          2.5,
	Bank:           -1.0,
}

// callTool connects the MCP server via in-memory transports and calls the tool.
func callTool(t *testing.T, pg internalmcp.PositionGetter, args map[string]any) *mcpsdk.CallToolResult {
	t.Helper()
	ctx := context.Background()

	srv := internalmcp.NewServer(pg)
	st, ct := mcpsdk.NewInMemoryTransports()

	_, err := srv.Connect(ctx, st)
	require.NoError(t, err)

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test", Version: "1.0"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	t.Cleanup(func() { cs.Close() })

	res, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "get_aircraft_position",
		Arguments: args,
	})
	require.NoError(t, err)
	return res
}

func TestGetAircraftPositionSuccess(t *testing.T) {
	pg := &mockPositionGetter{pos: samplePos}
	res := callTool(t, pg, nil)

	require.False(t, res.IsError)
	require.Len(t, res.Content, 1)

	text := res.Content[0].(*mcpsdk.TextContent).Text
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(text), &m))

	assert.InDelta(t, 47.6062, m["latitude"].(float64), 1e-9)
	assert.InDelta(t, -122.3321, m["longitude"].(float64), 1e-9)
	assert.InDelta(t, 35000.0, m["altitude_msl_ft"].(float64), 1e-9)
	assert.InDelta(t, 34950.0, m["altitude_agl_ft"].(float64), 1e-9)
	assert.InDelta(t, 270.0, m["heading_true_deg"].(float64), 1e-9)
	assert.InDelta(t, 268.5, m["heading_mag_deg"].(float64), 1e-9)
	assert.InDelta(t, 450.0, m["indicated_speed_kts"].(float64), 1e-9)
	assert.InDelta(t, 455.0, m["true_speed_kts"].(float64), 1e-9)
	assert.InDelta(t, 448.0, m["ground_speed_kts"].(float64), 1e-9)
	assert.InDelta(t, 500.0, m["vertical_speed_fpm"].(float64), 1e-9)

	// Validate RFC3339 timestamp
	ts, ok := m["timestamp"].(string)
	require.True(t, ok)
	_, err := time.Parse(time.RFC3339, ts)
	assert.NoError(t, err)
}

func TestGetAircraftPositionWithAttitude(t *testing.T) {
	pg := &mockPositionGetter{pos: samplePos}
	res := callTool(t, pg, map[string]any{"include_attitude": true})

	require.False(t, res.IsError)
	text := res.Content[0].(*mcpsdk.TextContent).Text
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(text), &m))

	assert.InDelta(t, 2.5, m["pitch_deg"].(float64), 1e-9, "pitch should be present when include_attitude=true")
	assert.InDelta(t, -1.0, m["bank_deg"].(float64), 1e-9, "bank should be present when include_attitude=true")
}

func TestGetAircraftPositionWithoutAttitude(t *testing.T) {
	pg := &mockPositionGetter{pos: samplePos}
	res := callTool(t, pg, map[string]any{"include_attitude": false})

	require.False(t, res.IsError)
	text := res.Content[0].(*mcpsdk.TextContent).Text
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(text), &m))

	_, hasPitch := m["pitch_deg"]
	_, hasBank := m["bank_deg"]
	assert.False(t, hasPitch, "pitch_deg should be omitted when include_attitude=false")
	assert.False(t, hasBank, "bank_deg should be omitted when include_attitude=false")
}

func TestGetAircraftPositionErrStale(t *testing.T) {
	pg := &mockPositionGetter{err: state.ErrStale}
	res := callTool(t, pg, nil)

	require.True(t, res.IsError)
	text := res.Content[0].(*mcpsdk.TextContent).Text
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(text), &m))

	assert.Equal(t, "DATA_STALE", m["code"])
	assert.Equal(t, true, m["recoverable"])
	assert.Equal(t, false, m["available"])
}

func TestGetAircraftPositionErrNotConnected(t *testing.T) {
	pg := &mockPositionGetter{err: simconnect.ErrNotConnected}
	res := callTool(t, pg, nil)

	require.True(t, res.IsError)
	text := res.Content[0].(*mcpsdk.TextContent).Text
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(text), &m))

	assert.Equal(t, "SIMULATOR_NOT_CONNECTED", m["code"])
	assert.Equal(t, true, m["recoverable"])
	assert.Equal(t, false, m["available"])
}

func TestGetAircraftPositionUnknownError(t *testing.T) {
	pg := &mockPositionGetter{err: errors.New("some unexpected error")}
	res := callTool(t, pg, nil)

	require.True(t, res.IsError)
	text := res.Content[0].(*mcpsdk.TextContent).Text
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(text), &m))

	assert.Equal(t, "UNKNOWN_ERROR", m["code"])
	assert.Equal(t, false, m["recoverable"])
	assert.Equal(t, false, m["available"])
}

func TestGetAircraftPositionTimestampIsRFC3339(t *testing.T) {
	pg := &mockPositionGetter{pos: samplePos}
	res := callTool(t, pg, nil)

	require.False(t, res.IsError)
	text := res.Content[0].(*mcpsdk.TextContent).Text
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(text), &m))

	ts := m["timestamp"].(string)
	parsed, err := time.Parse(time.RFC3339, ts)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().UTC(), parsed, 5*time.Second)
}
