package simconnect

import (
	"fmt"

	"github.com/eytandecker/flightsim-mcp/pkg/types"
)

const enginePayloadSize = 20 * 8 // 20 float64 fields × 8 bytes each

// ParseEnginePayload decodes a packed SimObjectData payload into EngineData.
// Expects exactly 160 bytes in EngineSimVars order.
func ParseEnginePayload(data []byte) (types.EngineData, error) {
	if len(data) < enginePayloadSize {
		return types.EngineData{}, fmt.Errorf("payload too short: got %d bytes, need %d", len(data), enginePayloadSize)
	}

	vals := make([]float64, len(EngineSimVars))
	for i := range EngineSimVars {
		offset := i * 8
		v, err := ParseSimVarValue(data[offset:offset+8], DataTypeFloat64)
		if err != nil {
			return types.EngineData{}, fmt.Errorf("parse simvar %d: %w", i, err)
		}
		vals[i] = v.(float64)
	}

	return types.EngineData{
		NumberOfEngines:   vals[0],
		ThrottlePosition1: vals[1],
		ThrottlePosition2: vals[2],
		RPM1:              vals[3],
		RPM2:              vals[4],
		N1Engine1:         vals[5],
		N1Engine2:         vals[6],
		N2Engine1:         vals[7],
		N2Engine2:         vals[8],
		FuelFlow1:         vals[9],
		FuelFlow2:         vals[10],
		EGT1:              vals[11],
		EGT2:              vals[12],
		OilTemp1:          vals[13],
		OilTemp2:          vals[14],
		OilPressure1:      vals[15],
		OilPressure2:      vals[16],
		FuelTotalQuantity: vals[17],
		FuelLeftQuantity:  vals[18],
		FuelRightQuantity: vals[19],
	}, nil
}
