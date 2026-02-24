package simconnect

import (
	"fmt"

	"github.com/eytandecker/flightsim-mcp/pkg/types"
)

const instrumentsPayloadSize = 11 * 8 // 11 float64 fields × 8 bytes each

// ParseInstrumentsPayload decodes a packed SimObjectData payload into FlightInstruments.
// Expects exactly 88 bytes in InstrumentsSimVars order.
func ParseInstrumentsPayload(data []byte) (types.FlightInstruments, error) {
	if len(data) < instrumentsPayloadSize {
		return types.FlightInstruments{}, fmt.Errorf("payload too short: got %d bytes, need %d", len(data), instrumentsPayloadSize)
	}

	vals := make([]float64, len(InstrumentsSimVars))
	for i := range InstrumentsSimVars {
		offset := i * 8
		v, err := ParseSimVarValue(data[offset:offset+8], DataTypeFloat64)
		if err != nil {
			return types.FlightInstruments{}, fmt.Errorf("parse simvar %d: %w", i, err)
		}
		vals[i] = v.(float64)
	}

	return types.FlightInstruments{
		IndicatedAltitude:   vals[0],
		KohlsmanSettingHg:   vals[1],
		VerticalSpeed:       vals[2],
		AirspeedIndicated:   vals[3],
		AirspeedTrue:        vals[4],
		AirspeedMach:        vals[5],
		HeadingIndicator:    vals[6],
		TurnIndicatorRate:   vals[7],
		TurnCoordinatorBall: vals[8],
		Pitch:               vals[9],
		Bank:                vals[10],
	}, nil
}
