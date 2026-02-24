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
		// Position (12)
		"PLANE LATITUDE", "PLANE LONGITUDE", "PLANE ALTITUDE", "PLANE ALT ABOVE GROUND",
		"PLANE HEADING DEGREES TRUE", "PLANE HEADING DEGREES MAGNETIC",
		"AIRSPEED INDICATED", "AIRSPEED TRUE", "GROUND VELOCITY",
		"VERTICAL SPEED", "PLANE PITCH DEGREES", "PLANE BANK DEGREES",
		// Instruments (6 new, 5 shared with position)
		"INDICATED ALTITUDE", "KOHLSMAN SETTING HG", "AIRSPEED MACH",
		"HEADING INDICATOR", "TURN INDICATOR RATE", "TURN COORDINATOR BALL",
		// Engine (20)
		"NUMBER OF ENGINES",
		"GENERAL ENG THROTTLE LEVER POSITION:1", "GENERAL ENG THROTTLE LEVER POSITION:2",
		"GENERAL ENG RPM:1", "GENERAL ENG RPM:2",
		"TURB ENG N1:1", "TURB ENG N1:2", "TURB ENG N2:1", "TURB ENG N2:2",
		"ENG FUEL FLOW GPH:1", "ENG FUEL FLOW GPH:2",
		"ENG EXHAUST GAS TEMPERATURE:1", "ENG EXHAUST GAS TEMPERATURE:2",
		"GENERAL ENG OIL TEMPERATURE:1", "GENERAL ENG OIL TEMPERATURE:2",
		"GENERAL ENG OIL PRESSURE:1", "GENERAL ENG OIL PRESSURE:2",
		"FUEL TOTAL QUANTITY", "FUEL LEFT QUANTITY", "FUEL RIGHT QUANTITY",
		// Environment (8)
		"AMBIENT WIND VELOCITY", "AMBIENT WIND DIRECTION", "AMBIENT TEMPERATURE",
		"AMBIENT PRESSURE", "AMBIENT VISIBILITY", "AMBIENT PRECIP STATE",
		"LOCAL TIME", "ZULU TIME",
		// Autopilot (12)
		"AUTOPILOT MASTER", "AUTOPILOT HEADING LOCK", "AUTOPILOT NAV1 LOCK",
		"AUTOPILOT APPROACH HOLD", "AUTOPILOT ALTITUDE LOCK", "AUTOPILOT VERTICAL HOLD",
		"AUTOPILOT AIRSPEED HOLD", "AUTOPILOT FLIGHT DIRECTOR ACTIVE",
		"AUTOPILOT HEADING LOCK DIR", "AUTOPILOT ALTITUDE LOCK VAR",
		"AUTOPILOT VERTICAL HOLD VAR", "AUTOPILOT AIRSPEED HOLD VAR",
	}
	for _, name := range expected {
		_, ok := registry.Get(name)
		assert.True(t, ok, "expected %q to be registered", name)
	}
}

func TestInstrumentsSimVars(t *testing.T) {
	assert.Len(t, InstrumentsSimVars, 11)
	assert.Equal(t, IndicatedAltitude, InstrumentsSimVars[0])
	assert.Equal(t, PlaneBank, InstrumentsSimVars[10])
}

func TestEngineSimVars(t *testing.T) {
	assert.Len(t, EngineSimVars, 20)
	assert.Equal(t, NumberOfEngines, EngineSimVars[0])
	assert.Equal(t, FuelRightQuantity, EngineSimVars[19])
}

func TestEnvironmentSimVars(t *testing.T) {
	assert.Len(t, EnvironmentSimVars, 8)
	assert.Equal(t, AmbientWindVelocity, EnvironmentSimVars[0])
	assert.Equal(t, ZuluTime, EnvironmentSimVars[7])
}

func TestAutopilotSimVars(t *testing.T) {
	assert.Len(t, AutopilotSimVars, 12)
	assert.Equal(t, APMaster, AutopilotSimVars[0])
	assert.Equal(t, APAirspeedHoldVar, AutopilotSimVars[11])
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
