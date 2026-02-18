package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/eytandecker/flightsim-mcp/internal/simconnect"
	"github.com/eytandecker/flightsim-mcp/internal/state"
	"github.com/eytandecker/flightsim-mcp/pkg/types"
)

// PositionGetter is the subset of state.Manager used by the MCP server.
type PositionGetter interface {
	GetPosition() (types.AircraftPosition, error)
}

// Server wraps the MCP SDK server and exposes SimConnect data as tools.
type Server struct {
	sdk   *mcpsdk.Server
	state PositionGetter
}

// NewServer creates a Server and registers the get_aircraft_position tool.
func NewServer(pg PositionGetter) *Server {
	s := &Server{
		sdk: mcpsdk.NewServer(&mcpsdk.Implementation{
			Name:    "flightsim-mcp",
			Version: "1.0.0",
		}, nil),
		state: pg,
	}

	tool := &mcpsdk.Tool{
		Name:        "get_aircraft_position",
		Description: "Returns live aircraft position, speed, and attitude data from Microsoft Flight Simulator 2024.",
	}
	mcpsdk.AddTool(s.sdk, tool, s.handleGetAircraftPosition)
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

// getPositionInput holds arguments for the get_aircraft_position tool.
type getPositionInput struct {
	IncludeAttitude bool `json:"include_attitude,omitempty"`
}

// AircraftPositionResponse is the JSON payload returned on success.
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

// SimulatorUnavailableResponse is returned when position data cannot be provided.
type SimulatorUnavailableResponse struct {
	Available   bool   `json:"available"`
	Error       string `json:"error"`
	Code        string `json:"code"`
	Recoverable bool   `json:"recoverable"`
	Suggestion  string `json:"suggestion"`
	Timestamp   string `json:"timestamp"`
}

func (s *Server) handleGetAircraftPosition(
	ctx context.Context,
	req *mcpsdk.CallToolRequest,
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
