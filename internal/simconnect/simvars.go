package simconnect

import (
	"encoding/binary"
	"fmt"
	"math"
)

// DataType represents the SimConnect data type for a SimVar value.
type DataType int

const (
	DataTypeFloat64 DataType = iota
	DataTypeInt32
)

// SimVarDef defines a SimConnect simulation variable.
type SimVarDef struct {
	Name     string
	Unit     string
	DataType DataType
	Size     int
}

// Predefined SimVar definitions.
var (
	PlaneLatitude = SimVarDef{
		Name:     "PLANE LATITUDE",
		Unit:     "degrees",
		DataType: DataTypeFloat64,
		Size:     8,
	}
	PlaneLongitude = SimVarDef{
		Name:     "PLANE LONGITUDE",
		Unit:     "degrees",
		DataType: DataTypeFloat64,
		Size:     8,
	}
	PlaneAltitude = SimVarDef{
		Name:     "PLANE ALTITUDE",
		Unit:     "feet",
		DataType: DataTypeFloat64,
		Size:     8,
	}
	PlaneAltAboveGround = SimVarDef{
		Name:     "PLANE ALT ABOVE GROUND",
		Unit:     "feet",
		DataType: DataTypeFloat64,
		Size:     8,
	}
	PlaneHeadingTrue = SimVarDef{
		Name:     "PLANE HEADING DEGREES TRUE",
		Unit:     "degrees",
		DataType: DataTypeFloat64,
		Size:     8,
	}
	PlaneHeadingMag = SimVarDef{
		Name:     "PLANE HEADING DEGREES MAGNETIC",
		Unit:     "degrees",
		DataType: DataTypeFloat64,
		Size:     8,
	}
	AirspeedIndicated = SimVarDef{
		Name:     "AIRSPEED INDICATED",
		Unit:     "knots",
		DataType: DataTypeFloat64,
		Size:     8,
	}
	AirspeedTrue = SimVarDef{
		Name:     "AIRSPEED TRUE",
		Unit:     "knots",
		DataType: DataTypeFloat64,
		Size:     8,
	}
	GroundVelocity = SimVarDef{
		Name:     "GROUND VELOCITY",
		Unit:     "knots",
		DataType: DataTypeFloat64,
		Size:     8,
	}
	VerticalSpeed = SimVarDef{
		Name:     "VERTICAL SPEED",
		Unit:     "feet/minute",
		DataType: DataTypeFloat64,
		Size:     8,
	}
	PlanePitch = SimVarDef{
		Name:     "PLANE PITCH DEGREES",
		Unit:     "degrees",
		DataType: DataTypeFloat64,
		Size:     8,
	}
	PlaneBank = SimVarDef{
		Name:     "PLANE BANK DEGREES",
		Unit:     "degrees",
		DataType: DataTypeFloat64,
		Size:     8,
	}
)

// SimVarRegistry holds the allowlist of valid SimVars.
type SimVarRegistry struct {
	vars map[string]SimVarDef
}

// NewSimVarRegistry creates a registry with all known SimVars.
func NewSimVarRegistry() *SimVarRegistry {
	r := &SimVarRegistry{
		vars: make(map[string]SimVarDef),
	}
	for _, v := range []SimVarDef{
		PlaneLatitude, PlaneLongitude, PlaneAltitude, PlaneAltAboveGround,
		PlaneHeadingTrue, PlaneHeadingMag, AirspeedIndicated, AirspeedTrue,
		GroundVelocity, VerticalSpeed, PlanePitch, PlaneBank,
	} {
		r.vars[v.Name] = v
	}
	return r
}

// Get returns the SimVarDef for the given name, if it exists.
func (r *SimVarRegistry) Get(name string) (SimVarDef, bool) {
	def, ok := r.vars[name]
	return def, ok
}

// Validate checks if a SimVar name is in the allowlist.
func (r *SimVarRegistry) Validate(name string) error {
	if _, ok := r.vars[name]; !ok {
		return fmt.Errorf("%w: %s", ErrInvalidSimVar, name)
	}
	return nil
}

// ParseSimVarValue decodes raw bytes into a typed value based on the DataType.
func ParseSimVarValue(data []byte, dt DataType) (any, error) {
	switch dt {
	case DataTypeFloat64:
		if len(data) < 8 {
			return nil, fmt.Errorf("float64 requires 8 bytes, got %d", len(data))
		}
		bits := binary.LittleEndian.Uint64(data[:8])
		return math.Float64frombits(bits), nil
	case DataTypeInt32:
		if len(data) < 4 {
			return nil, fmt.Errorf("int32 requires 4 bytes, got %d", len(data))
		}
		return int32(binary.LittleEndian.Uint32(data[:4])), nil //nolint:gosec // intentional reinterpretation of binary-encoded signed int32
	default:
		return nil, fmt.Errorf("unsupported data type: %d", dt)
	}
}
