package simconnect

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makePositionPayload(vals [12]float64) []byte {
	buf := make([]byte, 12*8)
	for i, v := range vals {
		binary.LittleEndian.PutUint64(buf[i*8:], math.Float64bits(v))
	}
	return buf
}

func TestParsePositionPayload(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
		check   func(t *testing.T, data []byte)
	}{
		{
			name: "valid 96-byte payload",
			data: makePositionPayload([12]float64{
				47.6062, -122.3321, 35000.0, 34950.0,
				270.0, 268.5, 450.0, 455.0, 448.0,
				500.0, 2.5, -1.0,
			}),
			check: func(t *testing.T, _ []byte) {},
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
			name:  "all-zero payload produces zero struct",
			data:  make([]byte, 96),
			check: func(t *testing.T, _ []byte) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, err := ParsePositionPayload(tt.data)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.name == "valid 96-byte payload" {
				assert.InDelta(t, 47.6062, pos.Latitude, 1e-9)
				assert.InDelta(t, -122.3321, pos.Longitude, 1e-9)
				assert.InDelta(t, 35000.0, pos.AltitudeMSL, 1e-9)
				assert.InDelta(t, 34950.0, pos.AltitudeAGL, 1e-9)
				assert.InDelta(t, 270.0, pos.HeadingTrue, 1e-9)
				assert.InDelta(t, 268.5, pos.HeadingMag, 1e-9)
				assert.InDelta(t, 450.0, pos.IndicatedSpeed, 1e-9)
				assert.InDelta(t, 455.0, pos.TrueSpeed, 1e-9)
				assert.InDelta(t, 448.0, pos.GroundSpeed, 1e-9)
				assert.InDelta(t, 500.0, pos.VerticalSpeed, 1e-9)
				assert.InDelta(t, 2.5, pos.Pitch, 1e-9)
				assert.InDelta(t, -1.0, pos.Bank, 1e-9)
			}
			if tt.name == "all-zero payload produces zero struct" {
				assert.Equal(t, 0.0, pos.Latitude)
				assert.Equal(t, 0.0, pos.Longitude)
				assert.Equal(t, 0.0, pos.AltitudeMSL)
			}
		})
	}
}
