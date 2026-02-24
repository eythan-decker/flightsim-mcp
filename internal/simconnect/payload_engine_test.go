package simconnect

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeEnginePayload(vals [20]float64) []byte { //nolint:gocritic
	buf := make([]byte, 20*8)
	for i, v := range vals {
		binary.LittleEndian.PutUint64(buf[i*8:], math.Float64bits(v))
	}
	return buf
}

func TestParseEnginePayload(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name: "valid 160-byte payload",
			data: makeEnginePayload([20]float64{
				2.0, 85.0, 85.0, 2400.0, 2400.0,
				92.0, 91.5, 98.0, 97.5, 120.0,
				118.0, 650.0, 648.0, 95.0, 94.0,
				55.0, 54.0, 500.0, 250.0, 250.0,
			}),
		},
		{
			name:    "truncated payload returns error",
			data:    make([]byte, 100),
			wantErr: true,
		},
		{
			name:    "empty payload returns error",
			data:    []byte{},
			wantErr: true,
		},
		{
			name: "all-zero payload produces zero struct",
			data: make([]byte, 160),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eng, err := ParseEnginePayload(tt.data)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.name == "valid 160-byte payload" {
				assert.InDelta(t, 2.0, eng.NumberOfEngines, 1e-9)
				assert.InDelta(t, 85.0, eng.ThrottlePosition1, 1e-9)
				assert.InDelta(t, 2400.0, eng.RPM1, 1e-9)
				assert.InDelta(t, 92.0, eng.N1Engine1, 1e-9)
				assert.InDelta(t, 120.0, eng.FuelFlow1, 1e-9)
				assert.InDelta(t, 650.0, eng.EGT1, 1e-9)
				assert.InDelta(t, 55.0, eng.OilPressure1, 1e-9)
				assert.InDelta(t, 500.0, eng.FuelTotalQuantity, 1e-9)
				assert.InDelta(t, 250.0, eng.FuelLeftQuantity, 1e-9)
				assert.InDelta(t, 250.0, eng.FuelRightQuantity, 1e-9)
			}
			if tt.name == "all-zero payload produces zero struct" {
				assert.Equal(t, 0.0, eng.NumberOfEngines)
				assert.Equal(t, 0.0, eng.FuelTotalQuantity)
			}
		})
	}
}
