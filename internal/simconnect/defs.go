package simconnect

// PositionSimVars is the ordered slice of SimVars used for the position data definition.
// The order determines the byte layout in SimObjectData responses.
var PositionSimVars = []SimVarDef{
	PlaneLatitude, PlaneLongitude, PlaneAltitude, PlaneAltAboveGround,
	PlaneHeadingTrue, PlaneHeadingMag, AirspeedIndicated, AirspeedTrue,
	GroundVelocity, VerticalSpeed, PlanePitch, PlaneBank,
}

const (
	DefIDPosition uint32 = 1
	ReqIDPosition uint32 = 1
	ObjectIDUser  uint32 = 0 // SIMCONNECT_OBJECT_ID_USER
)
