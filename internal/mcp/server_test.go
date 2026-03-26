package mcp_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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

// mockStateGetter controls what each Get* method returns in tests.
type mockStateGetter struct {
	pos  types.AircraftPosition
	inst types.FlightInstruments
	eng  types.EngineData
	env  types.Environment
	ap   types.AutopilotState
	err  error
}

func (m *mockStateGetter) GetPosition() (types.AircraftPosition, error) {
	return m.pos, m.err
}

func (m *mockStateGetter) GetInstruments() (types.FlightInstruments, error) {
	return m.inst, m.err
}

func (m *mockStateGetter) GetEngine() (types.EngineData, error) {
	return m.eng, m.err
}

func (m *mockStateGetter) GetEnvironment() (types.Environment, error) {
	return m.env, m.err
}

func (m *mockStateGetter) GetAutopilot() (types.AutopilotState, error) {
	return m.ap, m.err
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

var sampleInst = types.FlightInstruments{
	IndicatedAltitude:   35000.0,
	KohlsmanSettingHg:   29.92,
	VerticalSpeed:       500.0,
	AirspeedIndicated:   250.0,
	AirspeedTrue:        255.0,
	AirspeedMach:        0.78,
	HeadingIndicator:    270.0,
	TurnIndicatorRate:   0.02,
	TurnCoordinatorBall: 0.0,
	Pitch:               2.5,
	Bank:                -1.0,
}

var sampleEng = types.EngineData{
	NumberOfEngines:   2.0,
	ThrottlePosition1: 85.0,
	ThrottlePosition2: 85.0,
	RPM1:              2400.0,
	RPM2:              2400.0,
	N1Engine1:         92.0,
	N1Engine2:         91.5,
	FuelTotalQuantity: 500.0,
	FuelLeftQuantity:  250.0,
	FuelRightQuantity: 250.0,
}

var sampleEnv = types.Environment{
	WindVelocity:  15.0,
	WindDirection: 270.0,
	Temperature:   -5.5,
	Pressure:      29.92,
	Visibility:    10000.0,
	PrecipState:   4.0,
	LocalTime:     43200.0,
	ZuluTime:      50400.0,
}

var sampleAP = types.AutopilotState{
	Master:          1.0,
	HeadingLock:     1.0,
	Nav1Lock:        0.0,
	ApproachHold:    0.0,
	AltitudeLock:    1.0,
	VerticalHold:    1.0,
	AirspeedHold:    0.0,
	FlightDirector:  1.0,
	HeadingLockDir:  270.0,
	AltitudeLockVar: 35000.0,
	VerticalHoldVar: -500.0,
	AirspeedHoldVar: 250.0,
}

// callTool connects the MCP server via in-memory transports and calls the given tool.
func callTool(t *testing.T, sg internalmcp.StateGetter, toolName string, args map[string]any) *mcpsdk.CallToolResult {
	t.Helper()
	ctx := context.Background()

	srv := internalmcp.NewServer(sg)
	st, ct := mcpsdk.NewInMemoryTransports()

	_, err := srv.Connect(ctx, st)
	require.NoError(t, err)

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test", Version: "1.0"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	t.Cleanup(func() { cs.Close() })

	res, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      toolName,
		Arguments: args,
	})
	require.NoError(t, err)
	return res
}

func parseJSON(t *testing.T, res *mcpsdk.CallToolResult) map[string]any {
	t.Helper()
	require.Len(t, res.Content, 1)
	text := res.Content[0].(*mcpsdk.TextContent).Text
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(text), &m))
	return m
}

// --- get_aircraft_position tests ---

func TestGetAircraftPositionSuccess(t *testing.T) {
	sg := &mockStateGetter{pos: samplePos}
	res := callTool(t, sg, "get_aircraft_position", nil)

	require.False(t, res.IsError)
	m := parseJSON(t, res)

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

	ts, ok := m["timestamp"].(string)
	require.True(t, ok)
	_, err := time.Parse(time.RFC3339, ts)
	assert.NoError(t, err)
}

func TestGetAircraftPositionWithAttitude(t *testing.T) {
	sg := &mockStateGetter{pos: samplePos}
	res := callTool(t, sg, "get_aircraft_position", map[string]any{"include_attitude": true})

	require.False(t, res.IsError)
	m := parseJSON(t, res)

	assert.InDelta(t, 2.5, m["pitch_deg"].(float64), 1e-9)
	assert.InDelta(t, -1.0, m["bank_deg"].(float64), 1e-9)
}

func TestGetAircraftPositionWithoutAttitude(t *testing.T) {
	sg := &mockStateGetter{pos: samplePos}
	res := callTool(t, sg, "get_aircraft_position", map[string]any{"include_attitude": false})

	require.False(t, res.IsError)
	m := parseJSON(t, res)

	_, hasPitch := m["pitch_deg"]
	_, hasBank := m["bank_deg"]
	assert.False(t, hasPitch)
	assert.False(t, hasBank)
}

func TestGetAircraftPositionErrStale(t *testing.T) {
	sg := &mockStateGetter{err: state.ErrStale}
	res := callTool(t, sg, "get_aircraft_position", nil)

	require.True(t, res.IsError)
	m := parseJSON(t, res)

	assert.Equal(t, "DATA_STALE", m["code"])
	assert.Equal(t, true, m["recoverable"])
	assert.Equal(t, false, m["available"])
}

func TestGetAircraftPositionErrNotConnected(t *testing.T) {
	sg := &mockStateGetter{err: simconnect.ErrNotConnected}
	res := callTool(t, sg, "get_aircraft_position", nil)

	require.True(t, res.IsError)
	m := parseJSON(t, res)

	assert.Equal(t, "SIMULATOR_NOT_CONNECTED", m["code"])
	assert.Equal(t, true, m["recoverable"])
}

func TestGetAircraftPositionUnknownError(t *testing.T) {
	sg := &mockStateGetter{err: errors.New("some unexpected error")}
	res := callTool(t, sg, "get_aircraft_position", nil)

	require.True(t, res.IsError)
	m := parseJSON(t, res)

	assert.Equal(t, "UNKNOWN_ERROR", m["code"])
	assert.Equal(t, false, m["recoverable"])
}

func TestGetAircraftPositionTimestampIsRFC3339(t *testing.T) {
	sg := &mockStateGetter{pos: samplePos}
	res := callTool(t, sg, "get_aircraft_position", nil)

	require.False(t, res.IsError)
	m := parseJSON(t, res)

	ts := m["timestamp"].(string)
	parsed, err := time.Parse(time.RFC3339, ts)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().UTC(), parsed, 5*time.Second)
}

// --- get_flight_instruments tests ---

func TestGetFlightInstrumentsSuccess(t *testing.T) {
	sg := &mockStateGetter{inst: sampleInst}
	res := callTool(t, sg, "get_flight_instruments", nil)

	require.False(t, res.IsError)
	m := parseJSON(t, res)

	assert.InDelta(t, 35000.0, m["indicated_altitude_ft"].(float64), 1e-9)
	assert.InDelta(t, 29.92, m["kohlsman_setting_inhg"].(float64), 1e-9)
	assert.InDelta(t, 500.0, m["vertical_speed_fpm"].(float64), 1e-9)
	assert.InDelta(t, 250.0, m["airspeed_indicated_kts"].(float64), 1e-9)
	assert.InDelta(t, 0.78, m["airspeed_mach"].(float64), 1e-9)
	assert.InDelta(t, 270.0, m["heading_indicator_deg"].(float64), 1e-9)
	assert.InDelta(t, 2.5, m["pitch_deg"].(float64), 1e-9)
	assert.InDelta(t, -1.0, m["bank_deg"].(float64), 1e-9)
}

func TestGetFlightInstrumentsErrStale(t *testing.T) {
	sg := &mockStateGetter{err: state.ErrStale}
	res := callTool(t, sg, "get_flight_instruments", nil)

	require.True(t, res.IsError)
	m := parseJSON(t, res)
	assert.Equal(t, "DATA_STALE", m["code"])
}

// --- get_engine_data tests ---

func TestGetEngineDataSuccess(t *testing.T) {
	sg := &mockStateGetter{eng: sampleEng}
	res := callTool(t, sg, "get_engine_data", nil)

	require.False(t, res.IsError)
	m := parseJSON(t, res)

	assert.Equal(t, float64(2), m["number_of_engines"].(float64))
	assert.InDelta(t, 85.0, m["throttle_position_1_pct"].(float64), 1e-9)
	assert.InDelta(t, 2400.0, m["rpm_1"].(float64), 1e-9)
	assert.InDelta(t, 92.0, m["n1_engine_1_pct"].(float64), 1e-9)
	assert.InDelta(t, 500.0, m["fuel_total_gal"].(float64), 1e-9)
	assert.InDelta(t, 250.0, m["fuel_left_gal"].(float64), 1e-9)
}

func TestGetEngineDataErrStale(t *testing.T) {
	sg := &mockStateGetter{err: state.ErrStale}
	res := callTool(t, sg, "get_engine_data", nil)

	require.True(t, res.IsError)
	m := parseJSON(t, res)
	assert.Equal(t, "DATA_STALE", m["code"])
}

// --- get_environment tests ---

func TestGetEnvironmentSuccess(t *testing.T) {
	sg := &mockStateGetter{env: sampleEnv}
	res := callTool(t, sg, "get_environment", nil)

	require.False(t, res.IsError)
	m := parseJSON(t, res)

	assert.InDelta(t, 15.0, m["wind_velocity_kts"].(float64), 1e-9)
	assert.InDelta(t, 270.0, m["wind_direction_deg"].(float64), 1e-9)
	assert.InDelta(t, -5.5, m["temperature_celsius"].(float64), 1e-9)
	assert.InDelta(t, 29.92, m["pressure_inhg"].(float64), 1e-9)
	assert.InDelta(t, 10000.0, m["visibility_m"].(float64), 1e-9)
	assert.Equal(t, float64(4), m["precip_state"].(float64))
	assert.InDelta(t, 43200.0, m["local_time_sec"].(float64), 1e-9)
}

func TestGetEnvironmentErrStale(t *testing.T) {
	sg := &mockStateGetter{err: state.ErrStale}
	res := callTool(t, sg, "get_environment", nil)

	require.True(t, res.IsError)
	m := parseJSON(t, res)
	assert.Equal(t, "DATA_STALE", m["code"])
}

// --- get_autopilot_state tests ---

func TestGetAutopilotStateSuccess(t *testing.T) {
	sg := &mockStateGetter{ap: sampleAP}
	res := callTool(t, sg, "get_autopilot_state", nil)

	require.False(t, res.IsError)
	m := parseJSON(t, res)

	assert.Equal(t, true, m["master"])
	assert.Equal(t, true, m["heading_lock"])
	assert.Equal(t, false, m["nav1_lock"])
	assert.Equal(t, false, m["approach_hold"])
	assert.Equal(t, true, m["altitude_lock"])
	assert.Equal(t, true, m["vertical_hold"])
	assert.Equal(t, false, m["airspeed_hold"])
	assert.Equal(t, true, m["flight_director"])
	assert.InDelta(t, 270.0, m["heading_lock_dir_deg"].(float64), 1e-9)
	assert.InDelta(t, 35000.0, m["altitude_lock_var_ft"].(float64), 1e-9)
	assert.InDelta(t, -500.0, m["vertical_hold_var_fpm"].(float64), 1e-9)
	assert.InDelta(t, 250.0, m["airspeed_hold_var_kts"].(float64), 1e-9)
}

func TestGetAutopilotStateErrStale(t *testing.T) {
	sg := &mockStateGetter{err: state.ErrStale}
	res := callTool(t, sg, "get_autopilot_state", nil)

	require.True(t, res.IsError)
	m := parseJSON(t, res)
	assert.Equal(t, "DATA_STALE", m["code"])
}

func TestGetAutopilotStateBoolConversion(t *testing.T) {
	sg := &mockStateGetter{ap: types.AutopilotState{
		Master: 0.0, HeadingLock: 0.0, FlightDirector: 0.0,
	}}
	res := callTool(t, sg, "get_autopilot_state", nil)

	require.False(t, res.IsError)
	m := parseJSON(t, res)

	assert.Equal(t, false, m["master"])
	assert.Equal(t, false, m["heading_lock"])
	assert.Equal(t, false, m["flight_director"])
}

// --- HTTP handler tests ---

func TestHandler_ReturnsNonNil(t *testing.T) {
	sg := &mockStateGetter{}
	srv := internalmcp.NewServer(sg)
	assert.NotNil(t, srv.Handler())
}

func TestHealthHandler_Returns200(t *testing.T) {
	handler := internalmcp.HealthHandler()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", http.NoBody))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "ok")
}

type mockReadinessChecker struct {
	lastUpdated time.Time
}

func (m *mockReadinessChecker) LastUpdated() time.Time {
	return m.lastUpdated
}

func TestReadyHandler_Fresh(t *testing.T) {
	rc := &mockReadinessChecker{lastUpdated: time.Now()}
	handler := internalmcp.ReadyHandler(rc, 5*time.Second)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ready", http.NoBody))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "ready")
}

func TestReadyHandler_Stale(t *testing.T) {
	rc := &mockReadinessChecker{lastUpdated: time.Now().Add(-10 * time.Second)}
	handler := internalmcp.ReadyHandler(rc, 5*time.Second)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ready", http.NoBody))

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), "not ready")
}

func TestReadyHandler_NeverUpdated(t *testing.T) {
	rc := &mockReadinessChecker{lastUpdated: time.Time{}}
	handler := internalmcp.ReadyHandler(rc, 5*time.Second)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ready", http.NoBody))

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), "not ready")
}
