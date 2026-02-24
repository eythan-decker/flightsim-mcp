package simconnect

import (
	"fmt"

	"github.com/eytandecker/flightsim-mcp/pkg/types"
)

const environmentPayloadSize = 8 * 8 // 8 float64 fields × 8 bytes each

// ParseEnvironmentPayload decodes a packed SimObjectData payload into Environment.
// Expects exactly 64 bytes in EnvironmentSimVars order.
func ParseEnvironmentPayload(data []byte) (types.Environment, error) {
	if len(data) < environmentPayloadSize {
		return types.Environment{}, fmt.Errorf("payload too short: got %d bytes, need %d", len(data), environmentPayloadSize)
	}

	vals := make([]float64, len(EnvironmentSimVars))
	for i := range EnvironmentSimVars {
		offset := i * 8
		v, err := ParseSimVarValue(data[offset:offset+8], DataTypeFloat64)
		if err != nil {
			return types.Environment{}, fmt.Errorf("parse simvar %d: %w", i, err)
		}
		vals[i] = v.(float64)
	}

	return types.Environment{
		WindVelocity:  vals[0],
		WindDirection: vals[1],
		Temperature:   vals[2],
		Pressure:      vals[3],
		Visibility:    vals[4],
		PrecipState:   vals[5],
		LocalTime:     vals[6],
		ZuluTime:      vals[7],
	}, nil
}
