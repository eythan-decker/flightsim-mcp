package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/eytandecker/flightsim-mcp/internal/simconnect"
	"github.com/eytandecker/flightsim-mcp/internal/state"
	"github.com/eytandecker/flightsim-mcp/pkg/types"
)

// StateGetter is the subset of state.Manager used by the MCP server.
type StateGetter interface {
	GetPosition() (types.AircraftPosition, error)
	GetInstruments() (types.FlightInstruments, error)
	GetEngine() (types.EngineData, error)
	GetEnvironment() (types.Environment, error)
	GetAutopilot() (types.AutopilotState, error)
}

// Server wraps the MCP SDK server and exposes SimConnect data as tools.
type Server struct {
	sdk   *mcpsdk.Server
	state StateGetter
}

// NewServer creates a Server and registers all MCP tools.
func NewServer(sg StateGetter) *Server {
	s := &Server{
		sdk: mcpsdk.NewServer(&mcpsdk.Implementation{
			Name:    "flightsim-mcp",
			Version: "1.0.0",
		}, nil),
		state: sg,
	}

	mcpsdk.AddTool(s.sdk, &mcpsdk.Tool{
		Name:        "get_aircraft_position",
		Description: "Returns live aircraft position, speed, and attitude data from Microsoft Flight Simulator 2024.",
	}, s.handleGetAircraftPosition)

	mcpsdk.AddTool(s.sdk, &mcpsdk.Tool{
		Name:        "get_flight_instruments",
		Description: "Returns primary flight instrument readings including altimeter, airspeed, attitude, and heading indicators.",
	}, s.handleGetFlightInstruments)

	mcpsdk.AddTool(s.sdk, &mcpsdk.Tool{
		Name:        "get_engine_data",
		Description: "Returns engine performance data for up to 2 engines including RPM, N1/N2, temperatures, pressures, and fuel quantities.",
	}, s.handleGetEngineData)

	mcpsdk.AddTool(s.sdk, &mcpsdk.Tool{
		Name:        "get_environment",
		Description: "Returns weather and environment data including wind, temperature, pressure, visibility, and time.",
	}, s.handleGetEnvironment)

	mcpsdk.AddTool(s.sdk, &mcpsdk.Tool{
		Name:        "get_autopilot_state",
		Description: "Returns autopilot mode flags and target values including heading, altitude, vertical speed, and airspeed settings.",
	}, s.handleGetAutopilotState)

	return s
}

// Run starts the MCP server over stdio and blocks until ctx is done.
func (s *Server) Run(ctx context.Context) error {
	return s.sdk.Run(ctx, &mcpsdk.StdioTransport{})
}

// Connect connects the server to an existing transport (used in tests).
func (s *Server) Connect(ctx context.Context, t mcpsdk.Transport) (*mcpsdk.ServerSession, error) {
	return s.sdk.Connect(ctx, t, nil)
}

// Handler returns an http.Handler for MCP-over-HTTP (Streamable HTTP transport).
func (s *Server) Handler() http.Handler {
	return mcpsdk.NewStreamableHTTPHandler(
		func(_ *http.Request) *mcpsdk.Server { return s.sdk },
		&mcpsdk.StreamableHTTPOptions{Stateless: true},
	)
}

// ReadinessChecker provides the most recent data update time.
type ReadinessChecker interface {
	LastUpdated() time.Time
}

// HealthHandler returns a liveness probe handler (always 200 if process is alive).
func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck // best-effort response write
	}
}

// ReadyHandler returns a readiness probe handler.
// Returns 503 when no data has been received or data is older than staleThreshold.
func ReadyHandler(rc ReadinessChecker, staleThreshold time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		lu := rc.LastUpdated()
		if lu.IsZero() || time.Since(lu) > staleThreshold {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not ready"}`)) //nolint:errcheck // best-effort response write
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`)) //nolint:errcheck // best-effort response write
	}
}

// --- Input structs ---

type getPositionInput struct {
	IncludeAttitude bool `json:"include_attitude,omitempty"`
}

type emptyInput struct{}

// --- Response structs ---

// AircraftPositionResponse is the JSON payload returned by get_aircraft_position.
type AircraftPositionResponse struct {
	Latitude       float64  `json:"latitude"`
	Longitude      float64  `json:"longitude"`
	AltitudeMSL    float64  `json:"altitude_msl_ft"`
	AltitudeAGL    float64  `json:"altitude_agl_ft"`
	HeadingTrue    float64  `json:"heading_true_deg"`
	HeadingMag     float64  `json:"heading_mag_deg"`
	IndicatedSpeed float64  `json:"indicated_speed_kts"`
	TrueSpeed      float64  `json:"true_speed_kts"`
	GroundSpeed    float64  `json:"ground_speed_kts"`
	VerticalSpeed  float64  `json:"vertical_speed_fpm"`
	Pitch          *float64 `json:"pitch_deg,omitempty"`
	Bank           *float64 `json:"bank_deg,omitempty"`
	Timestamp      string   `json:"timestamp"`
}

// FlightInstrumentsResponse is the JSON payload returned by get_flight_instruments.
type FlightInstrumentsResponse struct {
	IndicatedAltitude   float64 `json:"indicated_altitude_ft"`
	KohlsmanSettingHg   float64 `json:"kohlsman_setting_inhg"`
	VerticalSpeed       float64 `json:"vertical_speed_fpm"`
	AirspeedIndicated   float64 `json:"airspeed_indicated_kts"`
	AirspeedTrue        float64 `json:"airspeed_true_kts"`
	AirspeedMach        float64 `json:"airspeed_mach"`
	HeadingIndicator    float64 `json:"heading_indicator_deg"`
	TurnIndicatorRate   float64 `json:"turn_indicator_rate_rps"`
	TurnCoordinatorBall float64 `json:"turn_coordinator_ball"`
	Pitch               float64 `json:"pitch_deg"`
	Bank                float64 `json:"bank_deg"`
	Timestamp           string  `json:"timestamp"`
}

// EngineDataResponse is the JSON payload returned by get_engine_data.
type EngineDataResponse struct {
	NumberOfEngines   int     `json:"number_of_engines"`
	ThrottlePosition1 float64 `json:"throttle_position_1_pct"`
	ThrottlePosition2 float64 `json:"throttle_position_2_pct"`
	RPM1              float64 `json:"rpm_1"`
	RPM2              float64 `json:"rpm_2"`
	N1Engine1         float64 `json:"n1_engine_1_pct"`
	N1Engine2         float64 `json:"n1_engine_2_pct"`
	N2Engine1         float64 `json:"n2_engine_1_pct"`
	N2Engine2         float64 `json:"n2_engine_2_pct"`
	FuelFlow1         float64 `json:"fuel_flow_1_gph"`
	FuelFlow2         float64 `json:"fuel_flow_2_gph"`
	EGT1              float64 `json:"egt_1_celsius"`
	EGT2              float64 `json:"egt_2_celsius"`
	OilTemp1          float64 `json:"oil_temp_1_celsius"`
	OilTemp2          float64 `json:"oil_temp_2_celsius"`
	OilPressure1      float64 `json:"oil_pressure_1_psi"`
	OilPressure2      float64 `json:"oil_pressure_2_psi"`
	FuelTotalQuantity float64 `json:"fuel_total_gal"`
	FuelLeftQuantity  float64 `json:"fuel_left_gal"`
	FuelRightQuantity float64 `json:"fuel_right_gal"`
	Timestamp         string  `json:"timestamp"`
}

// EnvironmentResponse is the JSON payload returned by get_environment.
type EnvironmentResponse struct {
	WindVelocity  float64 `json:"wind_velocity_kts"`
	WindDirection float64 `json:"wind_direction_deg"`
	Temperature   float64 `json:"temperature_celsius"`
	Pressure      float64 `json:"pressure_inhg"`
	Visibility    float64 `json:"visibility_m"`
	PrecipState   int     `json:"precip_state"`
	LocalTime     float64 `json:"local_time_sec"`
	ZuluTime      float64 `json:"zulu_time_sec"`
	Timestamp     string  `json:"timestamp"`
}

// AutopilotStateResponse is the JSON payload returned by get_autopilot_state.
type AutopilotStateResponse struct {
	Master          bool    `json:"master"`
	HeadingLock     bool    `json:"heading_lock"`
	Nav1Lock        bool    `json:"nav1_lock"`
	ApproachHold    bool    `json:"approach_hold"`
	AltitudeLock    bool    `json:"altitude_lock"`
	VerticalHold    bool    `json:"vertical_hold"`
	AirspeedHold    bool    `json:"airspeed_hold"`
	FlightDirector  bool    `json:"flight_director"`
	HeadingLockDir  float64 `json:"heading_lock_dir_deg"`
	AltitudeLockVar float64 `json:"altitude_lock_var_ft"`
	VerticalHoldVar float64 `json:"vertical_hold_var_fpm"`
	AirspeedHoldVar float64 `json:"airspeed_hold_var_kts"`
	Timestamp       string  `json:"timestamp"`
}

// SimulatorUnavailableResponse is returned when data cannot be provided.
type SimulatorUnavailableResponse struct {
	Available   bool   `json:"available"`
	Error       string `json:"error"`
	Code        string `json:"code"`
	Recoverable bool   `json:"recoverable"`
	Suggestion  string `json:"suggestion"`
	Timestamp   string `json:"timestamp"`
}

// --- Handlers ---

func (s *Server) handleGetAircraftPosition(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	input getPositionInput,
) (*mcpsdk.CallToolResult, any, error) {
	pos, err := s.state.GetPosition()
	if err != nil {
		return s.errorResult(err), nil, nil
	}

	resp := AircraftPositionResponse{
		Latitude:       pos.Latitude,
		Longitude:      pos.Longitude,
		AltitudeMSL:    pos.AltitudeMSL,
		AltitudeAGL:    pos.AltitudeAGL,
		HeadingTrue:    pos.HeadingTrue,
		HeadingMag:     pos.HeadingMag,
		IndicatedSpeed: pos.IndicatedSpeed,
		TrueSpeed:      pos.TrueSpeed,
		GroundSpeed:    pos.GroundSpeed,
		VerticalSpeed:  pos.VerticalSpeed,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
	}
	if input.IncludeAttitude {
		p, b := pos.Pitch, pos.Bank
		resp.Pitch = &p
		resp.Bank = &b
	}

	return s.jsonResult(resp)
}

func (s *Server) handleGetFlightInstruments(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	_ emptyInput,
) (*mcpsdk.CallToolResult, any, error) {
	inst, err := s.state.GetInstruments()
	if err != nil {
		return s.errorResult(err), nil, nil
	}

	resp := FlightInstrumentsResponse{
		IndicatedAltitude:   inst.IndicatedAltitude,
		KohlsmanSettingHg:   inst.KohlsmanSettingHg,
		VerticalSpeed:       inst.VerticalSpeed,
		AirspeedIndicated:   inst.AirspeedIndicated,
		AirspeedTrue:        inst.AirspeedTrue,
		AirspeedMach:        inst.AirspeedMach,
		HeadingIndicator:    inst.HeadingIndicator,
		TurnIndicatorRate:   inst.TurnIndicatorRate,
		TurnCoordinatorBall: inst.TurnCoordinatorBall,
		Pitch:               inst.Pitch,
		Bank:                inst.Bank,
		Timestamp:           time.Now().UTC().Format(time.RFC3339),
	}

	return s.jsonResult(resp)
}

func (s *Server) handleGetEngineData(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	_ emptyInput,
) (*mcpsdk.CallToolResult, any, error) {
	eng, err := s.state.GetEngine()
	if err != nil {
		return s.errorResult(err), nil, nil
	}

	resp := EngineDataResponse{
		NumberOfEngines:   int(eng.NumberOfEngines),
		ThrottlePosition1: eng.ThrottlePosition1,
		ThrottlePosition2: eng.ThrottlePosition2,
		RPM1:              eng.RPM1,
		RPM2:              eng.RPM2,
		N1Engine1:         eng.N1Engine1,
		N1Engine2:         eng.N1Engine2,
		N2Engine1:         eng.N2Engine1,
		N2Engine2:         eng.N2Engine2,
		FuelFlow1:         eng.FuelFlow1,
		FuelFlow2:         eng.FuelFlow2,
		EGT1:              eng.EGT1,
		EGT2:              eng.EGT2,
		OilTemp1:          eng.OilTemp1,
		OilTemp2:          eng.OilTemp2,
		OilPressure1:      eng.OilPressure1,
		OilPressure2:      eng.OilPressure2,
		FuelTotalQuantity: eng.FuelTotalQuantity,
		FuelLeftQuantity:  eng.FuelLeftQuantity,
		FuelRightQuantity: eng.FuelRightQuantity,
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
	}

	return s.jsonResult(resp)
}

func (s *Server) handleGetEnvironment(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	_ emptyInput,
) (*mcpsdk.CallToolResult, any, error) {
	env, err := s.state.GetEnvironment()
	if err != nil {
		return s.errorResult(err), nil, nil
	}

	resp := EnvironmentResponse{
		WindVelocity:  env.WindVelocity,
		WindDirection: env.WindDirection,
		Temperature:   env.Temperature,
		Pressure:      env.Pressure,
		Visibility:    env.Visibility,
		PrecipState:   int(env.PrecipState),
		LocalTime:     env.LocalTime,
		ZuluTime:      env.ZuluTime,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	}

	return s.jsonResult(resp)
}

func (s *Server) handleGetAutopilotState(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	_ emptyInput,
) (*mcpsdk.CallToolResult, any, error) {
	ap, err := s.state.GetAutopilot()
	if err != nil {
		return s.errorResult(err), nil, nil
	}

	resp := AutopilotStateResponse{
		Master:          ap.Master != 0,
		HeadingLock:     ap.HeadingLock != 0,
		Nav1Lock:        ap.Nav1Lock != 0,
		ApproachHold:    ap.ApproachHold != 0,
		AltitudeLock:    ap.AltitudeLock != 0,
		VerticalHold:    ap.VerticalHold != 0,
		AirspeedHold:    ap.AirspeedHold != 0,
		FlightDirector:  ap.FlightDirector != 0,
		HeadingLockDir:  ap.HeadingLockDir,
		AltitudeLockVar: ap.AltitudeLockVar,
		VerticalHoldVar: ap.VerticalHoldVar,
		AirspeedHoldVar: ap.AirspeedHoldVar,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
	}

	return s.jsonResult(resp)
}

// --- Helpers ---

func (s *Server) jsonResult(resp any) (*mcpsdk.CallToolResult, any, error) {
	data, err := json.Marshal(resp)
	if err != nil {
		return nil, nil, err
	}
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(data)}},
	}, nil, nil
}

func (s *Server) errorResult(err error) *mcpsdk.CallToolResult {
	resp := SimulatorUnavailableResponse{
		Available: false,
		Error:     err.Error(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	switch {
	case errors.Is(err, state.ErrStale):
		resp.Code = "DATA_STALE"
		resp.Recoverable = true
		resp.Suggestion = "Wait for the simulator to send fresh data."
	case errors.Is(err, simconnect.ErrNotConnected):
		resp.Code = "SIMULATOR_NOT_CONNECTED"
		resp.Recoverable = true
		resp.Suggestion = "Ensure Microsoft Flight Simulator is running."
	default:
		resp.Code = "UNKNOWN_ERROR"
		resp.Recoverable = false
		resp.Suggestion = "Check application logs for details."
	}

	data, _ := json.Marshal(resp)
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(data)}},
		IsError: true,
	}
}
