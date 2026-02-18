package simconnect

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSimVarValue(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		dt      DataType
		want    any
		wantErr bool
	}{
		{
			name: "float64 latitude",
			data: func() []byte {
				b := make([]byte, 8)
				binary.LittleEndian.PutUint64(b, math.Float64bits(47.6062))
				return b
			}(),
			dt:   DataTypeFloat64,
			want: 47.6062,
		},
		{
			name: "float64 negative longitude",
			data: func() []byte {
				b := make([]byte, 8)
				binary.LittleEndian.PutUint64(b, math.Float64bits(-122.3321))
				return b
			}(),
			dt:   DataTypeFloat64,
			want: -122.3321,
		},
		{
			name: "int32 boolean true",
			data: func() []byte {
				b := make([]byte, 4)
				binary.LittleEndian.PutUint32(b, 1)
				return b
			}(),
			dt:   DataTypeInt32,
			want: int32(1),
		},
		{
			name: "int32 zero",
			data: func() []byte {
				b := make([]byte, 4)
				binary.LittleEndian.PutUint32(b, 0)
				return b
			}(),
			dt:   DataTypeInt32,
			want: int32(0),
		},
		{
			name:    "float64 insufficient bytes",
			data:    make([]byte, 4),
			dt:      DataTypeFloat64,
			wantErr: true,
		},
		{
			name:    "int32 insufficient bytes",
			data:    make([]byte, 2),
			dt:      DataTypeInt32,
			wantErr: true,
		},
		{
			name:    "empty data",
			data:    nil,
			dt:      DataTypeFloat64,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSimVarValue(tt.data, tt.dt)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSimVarRegistry(t *testing.T) {
	registry := NewSimVarRegistry()

	t.Run("PlaneLatitude is registered", func(t *testing.T) {
		def, ok := registry.Get("PLANE LATITUDE")
		require.True(t, ok)
		assert.Equal(t, "PLANE LATITUDE", def.Name)
		assert.Equal(t, "degrees", def.Unit)
		assert.Equal(t, DataTypeFloat64, def.DataType)
		assert.Equal(t, 8, def.Size)
	})

	t.Run("invalid simvar rejected", func(t *testing.T) {
		_, ok := registry.Get("INVALID_VAR_NAME")
		assert.False(t, ok)
	})

	t.Run("validate returns error for unknown var", func(t *testing.T) {
		err := registry.Validate("NOT_A_REAL_VAR")
		assert.ErrorIs(t, err, ErrInvalidSimVar)
	})

	t.Run("validate succeeds for known var", func(t *testing.T) {
		err := registry.Validate("PLANE LATITUDE")
		assert.NoError(t, err)
	})
}

func TestPlaneLatitudeDefinition(t *testing.T) {
	assert.Equal(t, "PLANE LATITUDE", PlaneLatitude.Name)
	assert.Equal(t, "degrees", PlaneLatitude.Unit)
	assert.Equal(t, DataTypeFloat64, PlaneLatitude.DataType)
	assert.Equal(t, 8, PlaneLatitude.Size)
}

func TestAllSimVarsRegistered(t *testing.T) {
	registry := NewSimVarRegistry()
	expected := []string{
		"PLANE LATITUDE", "PLANE LONGITUDE", "PLANE ALTITUDE", "PLANE ALT ABOVE GROUND",
		"PLANE HEADING DEGREES TRUE", "PLANE HEADING DEGREES MAGNETIC",
		"AIRSPEED INDICATED", "AIRSPEED TRUE", "GROUND VELOCITY",
		"VERTICAL SPEED", "PLANE PITCH DEGREES", "PLANE BANK DEGREES",
	}
	for _, name := range expected {
		_, ok := registry.Get(name)
		assert.True(t, ok, "expected %q to be registered", name)
	}
}

func TestPositionSimVars(t *testing.T) {
	assert.Len(t, PositionSimVars, 12)
	assert.Equal(t, PlaneLatitude, PositionSimVars[0])
	assert.Equal(t, PlaneLongitude, PositionSimVars[1])
	assert.Equal(t, PlaneAltitude, PositionSimVars[2])
	assert.Equal(t, PlaneAltAboveGround, PositionSimVars[3])
	assert.Equal(t, PlaneHeadingTrue, PositionSimVars[4])
	assert.Equal(t, PlaneHeadingMag, PositionSimVars[5])
	assert.Equal(t, AirspeedIndicated, PositionSimVars[6])
	assert.Equal(t, AirspeedTrue, PositionSimVars[7])
	assert.Equal(t, GroundVelocity, PositionSimVars[8])
	assert.Equal(t, VerticalSpeed, PositionSimVars[9])
	assert.Equal(t, PlanePitch, PositionSimVars[10])
	assert.Equal(t, PlaneBank, PositionSimVars[11])
}
