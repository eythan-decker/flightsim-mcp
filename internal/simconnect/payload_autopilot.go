package simconnect

import (
	"fmt"

	"github.com/eytandecker/flightsim-mcp/pkg/types"
)

const autopilotPayloadSize = 12 * 8 // 12 float64 fields × 8 bytes each

// ParseAutopilotPayload decodes a packed SimObjectData payload into AutopilotState.
// Expects exactly 96 bytes in AutopilotSimVars order.
func ParseAutopilotPayload(data []byte) (types.AutopilotState, error) {
	if len(data) < autopilotPayloadSize {
		return types.AutopilotState{}, fmt.Errorf("payload too short: got %d bytes, need %d", len(data), autopilotPayloadSize)
	}

	vals := make([]float64, len(AutopilotSimVars))
	for i := range AutopilotSimVars {
		offset := i * 8
		v, err := ParseSimVarValue(data[offset:offset+8], DataTypeFloat64)
		if err != nil {
			return types.AutopilotState{}, fmt.Errorf("parse simvar %d: %w", i, err)
		}
		vals[i] = v.(float64)
	}

	return types.AutopilotState{
		Master:          vals[0],
		HeadingLock:     vals[1],
		Nav1Lock:        vals[2],
		ApproachHold:    vals[3],
		AltitudeLock:    vals[4],
		VerticalHold:    vals[5],
		AirspeedHold:    vals[6],
		FlightDirector:  vals[7],
		HeadingLockDir:  vals[8],
		AltitudeLockVar: vals[9],
		VerticalHoldVar: vals[10],
		AirspeedHoldVar: vals[11],
	}, nil
}
