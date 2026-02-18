package types

// AircraftPosition holds all position, speed, and attitude data for the user aircraft.
type AircraftPosition struct {
	Latitude      float64
	Longitude     float64
	AltitudeMSL   float64
	AltitudeAGL   float64
	HeadingTrue   float64
	HeadingMag    float64
	IndicatedSpeed float64
	TrueSpeed     float64
	GroundSpeed   float64
	VerticalSpeed float64
	Pitch         float64
	Bank          float64
}
