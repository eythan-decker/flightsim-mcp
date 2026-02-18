package state

import (
	"sync"
	"time"

	"github.com/eytandecker/flightsim-mcp/pkg/types"
)

// Manager holds a concurrent-safe cache of aircraft position state.
type Manager struct {
	mu             sync.RWMutex
	position       types.AircraftPosition
	lastUpdated    time.Time
	staleThreshold time.Duration
}

// NewManager creates a Manager with the given stale threshold.
// A zero threshold disables staleness checking.
func NewManager(staleThreshold time.Duration) *Manager {
	return &Manager{staleThreshold: staleThreshold}
}

// Update stores a new position value and records the current time.
func (m *Manager) Update(pos types.AircraftPosition) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.position = pos
	m.lastUpdated = time.Now()
}

// GetPosition returns the cached position, or ErrStale if no data has been
// received yet or the data age exceeds the stale threshold.
func (m *Manager) GetPosition() (types.AircraftPosition, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.lastUpdated.IsZero() {
		return types.AircraftPosition{}, ErrStale
	}
	if m.staleThreshold > 0 && time.Since(m.lastUpdated) > m.staleThreshold {
		return types.AircraftPosition{}, ErrStale
	}
	return m.position, nil
}

// LastUpdated returns the time of the most recent Update, or zero if never updated.
func (m *Manager) LastUpdated() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastUpdated
}
