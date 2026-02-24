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

	// Flight instruments
	IndicatedAltitude = SimVarDef{
		Name: "INDICATED ALTITUDE", Unit: "feet",
		DataType: DataTypeFloat64, Size: 8,
	}
	KohlsmanSettingHg = SimVarDef{
		Name: "KOHLSMAN SETTING HG", Unit: "inHg",
		DataType: DataTypeFloat64, Size: 8,
	}
	AirspeedMach = SimVarDef{
		Name: "AIRSPEED MACH", Unit: "mach",
		DataType: DataTypeFloat64, Size: 8,
	}
	HeadingIndicator = SimVarDef{
		Name: "HEADING INDICATOR", Unit: "degrees",
		DataType: DataTypeFloat64, Size: 8,
	}
	TurnIndicatorRate = SimVarDef{
		Name: "TURN INDICATOR RATE", Unit: "radians per second",
		DataType: DataTypeFloat64, Size: 8,
	}
	TurnCoordinatorBall = SimVarDef{
		Name: "TURN COORDINATOR BALL", Unit: "position 128",
		DataType: DataTypeFloat64, Size: 8,
	}

	// Engine data
	NumberOfEngines = SimVarDef{
		Name: "NUMBER OF ENGINES", Unit: "number",
		DataType: DataTypeFloat64, Size: 8,
	}
	ThrottlePosition1 = SimVarDef{
		Name: "GENERAL ENG THROTTLE LEVER POSITION:1", Unit: "percent",
		DataType: DataTypeFloat64, Size: 8,
	}
	ThrottlePosition2 = SimVarDef{
		Name: "GENERAL ENG THROTTLE LEVER POSITION:2", Unit: "percent",
		DataType: DataTypeFloat64, Size: 8,
	}
	EngRPM1 = SimVarDef{
		Name: "GENERAL ENG RPM:1", Unit: "rpm",
		DataType: DataTypeFloat64, Size: 8,
	}
	EngRPM2 = SimVarDef{
		Name: "GENERAL ENG RPM:2", Unit: "rpm",
		DataType: DataTypeFloat64, Size: 8,
	}
	TurbEngN1_1 = SimVarDef{
		Name: "TURB ENG N1:1", Unit: "percent",
		DataType: DataTypeFloat64, Size: 8,
	}
	TurbEngN1_2 = SimVarDef{
		Name: "TURB ENG N1:2", Unit: "percent",
		DataType: DataTypeFloat64, Size: 8,
	}
	TurbEngN2_1 = SimVarDef{
		Name: "TURB ENG N2:1", Unit: "percent",
		DataType: DataTypeFloat64, Size: 8,
	}
	TurbEngN2_2 = SimVarDef{
		Name: "TURB ENG N2:2", Unit: "percent",
		DataType: DataTypeFloat64, Size: 8,
	}
	FuelFlow1 = SimVarDef{
		Name: "ENG FUEL FLOW GPH:1", Unit: "gallons per hour",
		DataType: DataTypeFloat64, Size: 8,
	}
	FuelFlow2 = SimVarDef{
		Name: "ENG FUEL FLOW GPH:2", Unit: "gallons per hour",
		DataType: DataTypeFloat64, Size: 8,
	}
	EGT1 = SimVarDef{
		Name: "ENG EXHAUST GAS TEMPERATURE:1", Unit: "celsius",
		DataType: DataTypeFloat64, Size: 8,
	}
	EGT2 = SimVarDef{
		Name: "ENG EXHAUST GAS TEMPERATURE:2", Unit: "celsius",
		DataType: DataTypeFloat64, Size: 8,
	}
	OilTemp1 = SimVarDef{
		Name: "GENERAL ENG OIL TEMPERATURE:1", Unit: "celsius",
		DataType: DataTypeFloat64, Size: 8,
	}
	OilTemp2 = SimVarDef{
		Name: "GENERAL ENG OIL TEMPERATURE:2", Unit: "celsius",
		DataType: DataTypeFloat64, Size: 8,
	}
	OilPressure1 = SimVarDef{
		Name: "GENERAL ENG OIL PRESSURE:1", Unit: "psi",
		DataType: DataTypeFloat64, Size: 8,
	}
	OilPressure2 = SimVarDef{
		Name: "GENERAL ENG OIL PRESSURE:2", Unit: "psi",
		DataType: DataTypeFloat64, Size: 8,
	}
	FuelTotalQuantity = SimVarDef{
		Name: "FUEL TOTAL QUANTITY", Unit: "gallons",
		DataType: DataTypeFloat64, Size: 8,
	}
	FuelLeftQuantity = SimVarDef{
		Name: "FUEL LEFT QUANTITY", Unit: "gallons",
		DataType: DataTypeFloat64, Size: 8,
	}
	FuelRightQuantity = SimVarDef{
		Name: "FUEL RIGHT QUANTITY", Unit: "gallons",
		DataType: DataTypeFloat64, Size: 8,
	}

	// Environment
	AmbientWindVelocity = SimVarDef{
		Name: "AMBIENT WIND VELOCITY", Unit: "knots",
		DataType: DataTypeFloat64, Size: 8,
	}
	AmbientWindDirection = SimVarDef{
		Name: "AMBIENT WIND DIRECTION", Unit: "degrees",
		DataType: DataTypeFloat64, Size: 8,
	}
	AmbientTemperature = SimVarDef{
		Name: "AMBIENT TEMPERATURE", Unit: "celsius",
		DataType: DataTypeFloat64, Size: 8,
	}
	AmbientPressure = SimVarDef{
		Name: "AMBIENT PRESSURE", Unit: "inHg",
		DataType: DataTypeFloat64, Size: 8,
	}
	AmbientVisibility = SimVarDef{
		Name: "AMBIENT VISIBILITY", Unit: "meters",
		DataType: DataTypeFloat64, Size: 8,
	}
	AmbientPrecipState = SimVarDef{
		Name: "AMBIENT PRECIP STATE", Unit: "mask",
		DataType: DataTypeFloat64, Size: 8,
	}
	LocalTime = SimVarDef{
		Name: "LOCAL TIME", Unit: "seconds",
		DataType: DataTypeFloat64, Size: 8,
	}
	ZuluTime = SimVarDef{
		Name: "ZULU TIME", Unit: "seconds",
		DataType: DataTypeFloat64, Size: 8,
	}

	// Autopilot
	APMaster = SimVarDef{
		Name: "AUTOPILOT MASTER", Unit: "bool",
		DataType: DataTypeFloat64, Size: 8,
	}
	APHeadingLock = SimVarDef{
		Name: "AUTOPILOT HEADING LOCK", Unit: "bool",
		DataType: DataTypeFloat64, Size: 8,
	}
	APNav1Lock = SimVarDef{
		Name: "AUTOPILOT NAV1 LOCK", Unit: "bool",
		DataType: DataTypeFloat64, Size: 8,
	}
	APApproachHold = SimVarDef{
		Name: "AUTOPILOT APPROACH HOLD", Unit: "bool",
		DataType: DataTypeFloat64, Size: 8,
	}
	APAltitudeLock = SimVarDef{
		Name: "AUTOPILOT ALTITUDE LOCK", Unit: "bool",
		DataType: DataTypeFloat64, Size: 8,
	}
	APVerticalHold = SimVarDef{
		Name: "AUTOPILOT VERTICAL HOLD", Unit: "bool",
		DataType: DataTypeFloat64, Size: 8,
	}
	APAirspeedHold = SimVarDef{
		Name: "AUTOPILOT AIRSPEED HOLD", Unit: "bool",
		DataType: DataTypeFloat64, Size: 8,
	}
	APFlightDirector = SimVarDef{
		Name: "AUTOPILOT FLIGHT DIRECTOR ACTIVE", Unit: "bool",
		DataType: DataTypeFloat64, Size: 8,
	}
	APHeadingLockDir = SimVarDef{
		Name: "AUTOPILOT HEADING LOCK DIR", Unit: "degrees",
		DataType: DataTypeFloat64, Size: 8,
	}
	APAltitudeLockVar = SimVarDef{
		Name: "AUTOPILOT ALTITUDE LOCK VAR", Unit: "feet",
		DataType: DataTypeFloat64, Size: 8,
	}
	APVerticalHoldVar = SimVarDef{
		Name: "AUTOPILOT VERTICAL HOLD VAR", Unit: "feet/minute",
		DataType: DataTypeFloat64, Size: 8,
	}
	APAirspeedHoldVar = SimVarDef{
		Name: "AUTOPILOT AIRSPEED HOLD VAR", Unit: "knots",
		DataType: DataTypeFloat64, Size: 8,
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
		// Position
		PlaneLatitude, PlaneLongitude, PlaneAltitude, PlaneAltAboveGround,
		PlaneHeadingTrue, PlaneHeadingMag, AirspeedIndicated, AirspeedTrue,
		GroundVelocity, VerticalSpeed, PlanePitch, PlaneBank,
		// Flight instruments
		IndicatedAltitude, KohlsmanSettingHg, AirspeedMach,
		HeadingIndicator, TurnIndicatorRate, TurnCoordinatorBall,
		// Engine data
		NumberOfEngines, ThrottlePosition1, ThrottlePosition2,
		EngRPM1, EngRPM2, TurbEngN1_1, TurbEngN1_2, TurbEngN2_1, TurbEngN2_2,
		FuelFlow1, FuelFlow2, EGT1, EGT2, OilTemp1, OilTemp2,
		OilPressure1, OilPressure2, FuelTotalQuantity, FuelLeftQuantity, FuelRightQuantity,
		// Environment
		AmbientWindVelocity, AmbientWindDirection, AmbientTemperature,
		AmbientPressure, AmbientVisibility, AmbientPrecipState, LocalTime, ZuluTime,
		// Autopilot
		APMaster, APHeadingLock, APNav1Lock, APApproachHold,
		APAltitudeLock, APVerticalHold, APAirspeedHold, APFlightDirector,
		APHeadingLockDir, APAltitudeLockVar, APVerticalHoldVar, APAirspeedHoldVar,
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
		return int32(binary.LittleEndian.Uint32(data[:4])), nil // #nosec G115 -- intentional reinterpretation of binary-encoded signed int32
	default:
		return nil, fmt.Errorf("unsupported data type: %d", dt)
	}
}
