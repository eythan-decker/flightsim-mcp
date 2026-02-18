package simconnect

import (
	"fmt"

	"github.com/eytandecker/flightsim-mcp/pkg/types"
)

const positionPayloadSize = 12 * 8 // 12 float64 fields × 8 bytes each

// ParsePositionPayload decodes a packed SimObjectData payload into an AircraftPosition.
// Expects exactly 96 bytes in PositionSimVars order.
func ParsePositionPayload(data []byte) (types.AircraftPosition, error) {
	if len(data) < positionPayloadSize {
		return types.AircraftPosition{}, fmt.Errorf("payload too short: got %d bytes, need %d", len(data), positionPayloadSize)
	}

	vals := make([]float64, len(PositionSimVars))
	for i := range PositionSimVars {
		offset := i * 8
		v, err := ParseSimVarValue(data[offset:offset+8], DataTypeFloat64)
		if err != nil {
			return types.AircraftPosition{}, fmt.Errorf("parse simvar %d: %w", i, err)
		}
		vals[i] = v.(float64)
	}

	return types.AircraftPosition{
		Latitude:       vals[0],
		Longitude:      vals[1],
		AltitudeMSL:    vals[2],
		AltitudeAGL:    vals[3],
		HeadingTrue:    vals[4],
		HeadingMag:     vals[5],
		IndicatedSpeed: vals[6],
		TrueSpeed:      vals[7],
		GroundSpeed:    vals[8],
		VerticalSpeed:  vals[9],
		Pitch:          vals[10],
		Bank:           vals[11],
	}, nil
}
