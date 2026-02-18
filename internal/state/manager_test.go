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
