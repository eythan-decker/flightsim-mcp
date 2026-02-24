package types

// FlightInstruments holds primary flight instrument readings.
type FlightInstruments struct {
	IndicatedAltitude   float64
	KohlsmanSettingHg   float64
	VerticalSpeed       float64
	AirspeedIndicated   float64
	AirspeedTrue        float64
	AirspeedMach        float64
	HeadingIndicator    float64
	TurnIndicatorRate   float64
	TurnCoordinatorBall float64
	Pitch               float64
	Bank                float64
}
