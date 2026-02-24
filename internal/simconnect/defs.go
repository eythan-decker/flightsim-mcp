package simconnect

// Ordered SimVar slices for each data group.
// The order determines the byte layout in SimObjectData responses.
var (
	PositionSimVars = []SimVarDef{
		PlaneLatitude, PlaneLongitude, PlaneAltitude, PlaneAltAboveGround,
		PlaneHeadingTrue, PlaneHeadingMag, AirspeedIndicated, AirspeedTrue,
		GroundVelocity, VerticalSpeed, PlanePitch, PlaneBank,
	}

	InstrumentsSimVars = []SimVarDef{
		IndicatedAltitude, KohlsmanSettingHg, VerticalSpeed, AirspeedIndicated,
		AirspeedTrue, AirspeedMach, HeadingIndicator, TurnIndicatorRate,
		TurnCoordinatorBall, PlanePitch, PlaneBank,
	}

	EngineSimVars = []SimVarDef{
		NumberOfEngines, ThrottlePosition1, ThrottlePosition2,
		EngRPM1, EngRPM2, TurbEngN1_1, TurbEngN1_2, TurbEngN2_1, TurbEngN2_2,
		FuelFlow1, FuelFlow2, EGT1, EGT2, OilTemp1, OilTemp2,
		OilPressure1, OilPressure2, FuelTotalQuantity, FuelLeftQuantity, FuelRightQuantity,
	}

	EnvironmentSimVars = []SimVarDef{
		AmbientWindVelocity, AmbientWindDirection, AmbientTemperature,
		AmbientPressure, AmbientVisibility, AmbientPrecipState, LocalTime, ZuluTime,
	}

	AutopilotSimVars = []SimVarDef{
		APMaster, APHeadingLock, APNav1Lock, APApproachHold,
		APAltitudeLock, APVerticalHold, APAirspeedHold, APFlightDirector,
		APHeadingLockDir, APAltitudeLockVar, APVerticalHoldVar, APAirspeedHoldVar,
	}
)

const (
	DefIDPosition    uint32 = 1
	ReqIDPosition    uint32 = 1
	DefIDInstruments uint32 = 2
	ReqIDInstruments uint32 = 2
	DefIDEngine      uint32 = 3
	ReqIDEngine      uint32 = 3
	DefIDEnvironment uint32 = 4
	ReqIDEnvironment uint32 = 4
	DefIDAutopilot   uint32 = 5
	ReqIDAutopilot   uint32 = 5
	ObjectIDUser     uint32 = 0 // SIMCONNECT_OBJECT_ID_USER
)
