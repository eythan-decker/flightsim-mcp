package state

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eytandecker/flightsim-mcp/pkg/types"
)

func samplePosition() types.AircraftPosition {
	return types.AircraftPosition{
		Latitude:       47.6062,
		Longitude:      -122.3321,
		AltitudeMSL:    35000.0,
		AltitudeAGL:    34950.0,
		HeadingTrue:    270.0,
		HeadingMag:     268.5,
		IndicatedSpeed: 450.0,
		TrueSpeed:      455.0,
		GroundSpeed:    448.0,
		VerticalSpeed:  500.0,
		Pitch:          2.5,
		Bank:           -1.0,
	}
}

func sampleInstruments() types.FlightInstruments {
	return types.FlightInstruments{
		IndicatedAltitude:   35000.0,
		KohlsmanSettingHg:   29.92,
		VerticalSpeed:       500.0,
		AirspeedIndicated:   250.0,
		AirspeedTrue:        255.0,
		AirspeedMach:        0.78,
		HeadingIndicator:    270.0,
		TurnIndicatorRate:   0.02,
		TurnCoordinatorBall: 0.0,
		Pitch:               2.5,
		Bank:                -1.0,
	}
}

func sampleEngine() types.EngineData {
	return types.EngineData{
		NumberOfEngines:   2.0,
		ThrottlePosition1: 85.0,
		ThrottlePosition2: 85.0,
		RPM1:              2400.0,
		RPM2:              2400.0,
		N1Engine1:         92.0,
		N1Engine2:         91.5,
		FuelTotalQuantity: 500.0,
	}
}

func sampleEnvironment() types.Environment {
	return types.Environment{
		WindVelocity:  15.0,
		WindDirection: 270.0,
		Temperature:   -5.5,
		Pressure:      29.92,
		Visibility:    10000.0,
		PrecipState:   0.0,
		LocalTime:     43200.0,
		ZuluTime:      50400.0,
	}
}

func sampleAutopilot() types.AutopilotState {
	return types.AutopilotState{
		Master:          1.0,
		HeadingLock:     1.0,
		AltitudeLock:    1.0,
		HeadingLockDir:  270.0,
		AltitudeLockVar: 35000.0,
		VerticalHoldVar: -500.0,
		AirspeedHoldVar: 250.0,
	}
}

// Position tests

func TestGetPositionReturnsStaleBeforeUpdate(t *testing.T) {
	mgr := NewManager(5 * time.Second)
	_, err := mgr.GetPosition()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrStale)
}

func TestUpdateAndGetPosition(t *testing.T) {
	mgr := NewManager(5 * time.Second)
	pos := samplePosition()
	mgr.Update(pos)

	got, err := mgr.GetPosition()
	require.NoError(t, err)
	assert.Equal(t, pos, got)
}

func TestGetPositionReturnsStaleWhenExpired(t *testing.T) {
	mgr := NewManager(1 * time.Millisecond)
	mgr.Update(samplePosition())

	time.Sleep(5 * time.Millisecond)

	_, err := mgr.GetPosition()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrStale)
}

func TestZeroThresholdNeverStale(t *testing.T) {
	mgr := NewManager(0)
	mgr.Update(samplePosition())

	time.Sleep(5 * time.Millisecond)

	_, err := mgr.GetPosition()
	assert.NoError(t, err)
}

func TestLastUpdatedZeroBeforeUpdate(t *testing.T) {
	mgr := NewManager(5 * time.Second)
	assert.True(t, mgr.LastUpdated().IsZero())
}

func TestLastUpdatedNonZeroAfterUpdate(t *testing.T) {
	mgr := NewManager(5 * time.Second)
	before := time.Now()
	mgr.Update(samplePosition())
	after := time.Now()

	lu := mgr.LastUpdated()
	assert.False(t, lu.IsZero())
	assert.True(t, !lu.Before(before) && !lu.After(after))
}

func TestConcurrentUpdateAndGetPosition(t *testing.T) {
	mgr := NewManager(5 * time.Second)
	pos := samplePosition()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			mgr.Update(pos)
		}()
		go func() {
			defer wg.Done()
			_, _ = mgr.GetPosition()
		}()
	}
	wg.Wait()
}

// Instruments tests

func TestGetInstrumentsReturnsStaleBeforeUpdate(t *testing.T) {
	mgr := NewManager(5 * time.Second)
	_, err := mgr.GetInstruments()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrStale)
}

func TestUpdateAndGetInstruments(t *testing.T) {
	mgr := NewManager(5 * time.Second)
	inst := sampleInstruments()
	mgr.UpdateInstruments(inst)

	got, err := mgr.GetInstruments()
	require.NoError(t, err)
	assert.Equal(t, inst, got)
}

func TestGetInstrumentsReturnsStaleWhenExpired(t *testing.T) {
	mgr := NewManager(1 * time.Millisecond)
	mgr.UpdateInstruments(sampleInstruments())
	time.Sleep(5 * time.Millisecond)

	_, err := mgr.GetInstruments()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrStale)
}

// Engine tests

func TestGetEngineReturnsStaleBeforeUpdate(t *testing.T) {
	mgr := NewManager(5 * time.Second)
	_, err := mgr.GetEngine()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrStale)
}

func TestUpdateAndGetEngine(t *testing.T) {
	mgr := NewManager(5 * time.Second)
	eng := sampleEngine()
	mgr.UpdateEngine(eng)

	got, err := mgr.GetEngine()
	require.NoError(t, err)
	assert.Equal(t, eng, got)
}

func TestGetEngineReturnsStaleWhenExpired(t *testing.T) {
	mgr := NewManager(1 * time.Millisecond)
	mgr.UpdateEngine(sampleEngine())
	time.Sleep(5 * time.Millisecond)

	_, err := mgr.GetEngine()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrStale)
}

// Environment tests

func TestGetEnvironmentReturnsStaleBeforeUpdate(t *testing.T) {
	mgr := NewManager(5 * time.Second)
	_, err := mgr.GetEnvironment()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrStale)
}

func TestUpdateAndGetEnvironment(t *testing.T) {
	mgr := NewManager(5 * time.Second)
	env := sampleEnvironment()
	mgr.UpdateEnvironment(env)

	got, err := mgr.GetEnvironment()
	require.NoError(t, err)
	assert.Equal(t, env, got)
}

func TestGetEnvironmentReturnsStaleWhenExpired(t *testing.T) {
	mgr := NewManager(1 * time.Millisecond)
	mgr.UpdateEnvironment(sampleEnvironment())
	time.Sleep(5 * time.Millisecond)

	_, err := mgr.GetEnvironment()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrStale)
}

// Autopilot tests

func TestGetAutopilotReturnsStaleBeforeUpdate(t *testing.T) {
	mgr := NewManager(5 * time.Second)
	_, err := mgr.GetAutopilot()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrStale)
}

func TestUpdateAndGetAutopilot(t *testing.T) {
	mgr := NewManager(5 * time.Second)
	ap := sampleAutopilot()
	mgr.UpdateAutopilot(ap)

	got, err := mgr.GetAutopilot()
	require.NoError(t, err)
	assert.Equal(t, ap, got)
}

func TestGetAutopilotReturnsStaleWhenExpired(t *testing.T) {
	mgr := NewManager(1 * time.Millisecond)
	mgr.UpdateAutopilot(sampleAutopilot())
	time.Sleep(5 * time.Millisecond)

	_, err := mgr.GetAutopilot()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrStale)
}

// Cross-group staleness independence

func TestCrossGroupStalenessIndependence(t *testing.T) {
	mgr := NewManager(1 * time.Millisecond)
	mgr.Update(samplePosition())
	time.Sleep(5 * time.Millisecond)

	// Position should be stale
	_, err := mgr.GetPosition()
	assert.ErrorIs(t, err, ErrStale)

	// Instruments updated fresh — should not be stale
	mgr.UpdateInstruments(sampleInstruments())
	_, err = mgr.GetInstruments()
	assert.NoError(t, err)
}

// Concurrent access across all groups

func TestConcurrentMultiGroupAccess(t *testing.T) {
	mgr := NewManager(5 * time.Second)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(5)
		go func() { defer wg.Done(); mgr.Update(samplePosition()) }()
		go func() { defer wg.Done(); mgr.UpdateInstruments(sampleInstruments()) }()
		go func() { defer wg.Done(); mgr.UpdateEngine(sampleEngine()) }()
		go func() { defer wg.Done(); mgr.UpdateEnvironment(sampleEnvironment()) }()
		go func() { defer wg.Done(); mgr.UpdateAutopilot(sampleAutopilot()) }()
	}
	wg.Wait()

	// All groups should be readable
	_, err := mgr.GetPosition()
	assert.NoError(t, err)
	_, err = mgr.GetInstruments()
	assert.NoError(t, err)
	_, err = mgr.GetEngine()
	assert.NoError(t, err)
	_, err = mgr.GetEnvironment()
	assert.NoError(t, err)
	_, err = mgr.GetAutopilot()
	assert.NoError(t, err)
}
