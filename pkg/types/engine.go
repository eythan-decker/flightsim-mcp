package types

// EngineData holds engine performance and fuel data for up to 2 engines.
type EngineData struct {
	NumberOfEngines   float64
	ThrottlePosition1 float64
	ThrottlePosition2 float64
	RPM1              float64
	RPM2              float64
	N1Engine1         float64
	N1Engine2         float64
	N2Engine1         float64
	N2Engine2         float64
	FuelFlow1         float64
	FuelFlow2         float64
	EGT1              float64
	EGT2              float64
	OilTemp1          float64
	OilTemp2          float64
	OilPressure1      float64
	OilPressure2      float64
	FuelTotalQuantity float64
	FuelLeftQuantity  float64
	FuelRightQuantity float64
}
