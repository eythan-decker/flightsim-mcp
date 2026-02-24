package types

// Environment holds weather and time-of-day data from the simulator.
type Environment struct {
	WindVelocity  float64
	WindDirection float64
	Temperature   float64
	Pressure      float64
	Visibility    float64
	PrecipState   float64
	LocalTime     float64
	ZuluTime      float64
}
