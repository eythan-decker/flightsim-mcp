package types

// AutopilotState holds autopilot mode flags and target values.
type AutopilotState struct {
	Master          float64
	HeadingLock     float64
	Nav1Lock        float64
	ApproachHold    float64
	AltitudeLock    float64
	VerticalHold    float64
	AirspeedHold    float64
	FlightDirector  float64
	HeadingLockDir  float64
	AltitudeLockVar float64
	VerticalHoldVar float64
	AirspeedHoldVar float64
}
