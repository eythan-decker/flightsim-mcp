package simconnect

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeInstrumentsPayload(vals [11]float64) []byte { //nolint:gocritic
	buf := make([]byte, 11*8)
	for i, v := range vals {
		binary.LittleEndian.PutUint64(buf[i*8:], math.Float64bits(v))
	}
	return buf
}

func TestParseInstrumentsPayload(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name: "valid 88-byte payload",
			data: makeInstrumentsPayload([11]float64{
				35000.0, 29.92, 500.0, 250.0, 255.0,
				0.78, 270.0, 0.02, 0.0, 2.5, -1.0,
			}),
		},
		{
			name:    "truncated payload returns error",
			data:    make([]byte, 50),
			wantErr: true,
		},
		{
			name:    "empty payload returns error",
			data:    []byte{},
			wantErr: true,
		},
		{
			name: "all-zero payload produces zero struct",
			data: make([]byte, 88),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst, err := ParseInstrumentsPayload(tt.data)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.name == "valid 88-byte payload" {
				assert.InDelta(t, 35000.0, inst.IndicatedAltitude, 1e-9)
				assert.InDelta(t, 29.92, inst.KohlsmanSettingHg, 1e-9)
				assert.InDelta(t, 500.0, inst.VerticalSpeed, 1e-9)
				assert.InDelta(t, 250.0, inst.AirspeedIndicated, 1e-9)
				assert.InDelta(t, 0.78, inst.AirspeedMach, 1e-9)
				assert.InDelta(t, 270.0, inst.HeadingIndicator, 1e-9)
				assert.InDelta(t, 2.5, inst.Pitch, 1e-9)
				assert.InDelta(t, -1.0, inst.Bank, 1e-9)
			}
			if tt.name == "all-zero payload produces zero struct" {
				assert.Equal(t, 0.0, inst.IndicatedAltitude)
				assert.Equal(t, 0.0, inst.AirspeedMach)
			}
		})
	}
}
