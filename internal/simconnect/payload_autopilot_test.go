package simconnect

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeAutopilotPayload(vals [12]float64) []byte { //nolint:gocritic
	buf := make([]byte, 12*8)
	for i, v := range vals {
		binary.LittleEndian.PutUint64(buf[i*8:], math.Float64bits(v))
	}
	return buf
}

func TestParseAutopilotPayload(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name: "valid 96-byte payload",
			data: makeAutopilotPayload([12]float64{
				1.0, 1.0, 0.0, 0.0, 1.0, 1.0,
				0.0, 1.0, 270.0, 35000.0, -500.0, 250.0,
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
			data: make([]byte, 96),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ap, err := ParseAutopilotPayload(tt.data)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.name == "valid 96-byte payload" {
				assert.InDelta(t, 1.0, ap.Master, 1e-9)
				assert.InDelta(t, 1.0, ap.HeadingLock, 1e-9)
				assert.InDelta(t, 0.0, ap.Nav1Lock, 1e-9)
				assert.InDelta(t, 1.0, ap.AltitudeLock, 1e-9)
				assert.InDelta(t, 1.0, ap.FlightDirector, 1e-9)
				assert.InDelta(t, 270.0, ap.HeadingLockDir, 1e-9)
				assert.InDelta(t, 35000.0, ap.AltitudeLockVar, 1e-9)
				assert.InDelta(t, -500.0, ap.VerticalHoldVar, 1e-9)
				assert.InDelta(t, 250.0, ap.AirspeedHoldVar, 1e-9)
			}
			if tt.name == "all-zero payload produces zero struct" {
				assert.Equal(t, 0.0, ap.Master)
				assert.Equal(t, 0.0, ap.HeadingLockDir)
			}
		})
	}
}
