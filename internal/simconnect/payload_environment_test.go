package simconnect

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeEnvironmentPayload(vals [8]float64) []byte { //nolint:gocritic
	buf := make([]byte, 8*8)
	for i, v := range vals {
		binary.LittleEndian.PutUint64(buf[i*8:], math.Float64bits(v))
	}
	return buf
}

func TestParseEnvironmentPayload(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name: "valid 64-byte payload",
			data: makeEnvironmentPayload([8]float64{
				15.0, 270.0, -5.5, 29.92, 10000.0, 4.0, 43200.0, 50400.0,
			}),
		},
		{
			name:    "truncated payload returns error",
			data:    make([]byte, 30),
			wantErr: true,
		},
		{
			name:    "empty payload returns error",
			data:    []byte{},
			wantErr: true,
		},
		{
			name: "all-zero payload produces zero struct",
			data: make([]byte, 64),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, err := ParseEnvironmentPayload(tt.data)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.name == "valid 64-byte payload" {
				assert.InDelta(t, 15.0, env.WindVelocity, 1e-9)
				assert.InDelta(t, 270.0, env.WindDirection, 1e-9)
				assert.InDelta(t, -5.5, env.Temperature, 1e-9)
				assert.InDelta(t, 29.92, env.Pressure, 1e-9)
				assert.InDelta(t, 10000.0, env.Visibility, 1e-9)
				assert.InDelta(t, 4.0, env.PrecipState, 1e-9)
				assert.InDelta(t, 43200.0, env.LocalTime, 1e-9)
				assert.InDelta(t, 50400.0, env.ZuluTime, 1e-9)
			}
			if tt.name == "all-zero payload produces zero struct" {
				assert.Equal(t, 0.0, env.WindVelocity)
				assert.Equal(t, 0.0, env.Temperature)
			}
		})
	}
}
