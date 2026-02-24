package state

import (
	"sync"
	"time"

	"github.com/eytandecker/flightsim-mcp/pkg/types"
)

// Data group keys for per-group staleness tracking.
const (
	GroupPosition    = "position"
	GroupInstruments = "instruments"
	GroupEngine      = "engine"
	GroupEnvironment = "environment"
	GroupAutopilot   = "autopilot"
)

// Manager holds a concurrent-safe cache of all aircraft state data.
type Manager struct {
	mu             sync.RWMutex
	position       types.AircraftPosition
	instruments    types.FlightInstruments
	engine         types.EngineData
	environment    types.Environment
	autopilot      types.AutopilotState
	lastUpdated    map[string]time.Time
	staleThreshold time.Duration
}

// NewManager creates a Manager with the given stale threshold.
// A zero threshold disables staleness checking.
func NewManager(staleThreshold time.Duration) *Manager {
	return &Manager{
		staleThreshold: staleThreshold,
		lastUpdated:    make(map[string]time.Time),
	}
}

// isStale checks whether the given group's data is stale. Caller must hold at least RLock.
func (m *Manager) isStale(group string) bool {
	lu, ok := m.lastUpdated[group]
	if !ok || lu.IsZero() {
		return true
	}
	if m.staleThreshold > 0 && time.Since(lu) > m.staleThreshold {
		return true
	}
	return false
}

// Update stores a new position value.
func (m *Manager) Update(pos types.AircraftPosition) { //nolint:gocritic
	m.mu.Lock()
	defer m.mu.Unlock()
	m.position = pos
	m.lastUpdated[GroupPosition] = time.Now()
}

// GetPosition returns the cached position, or ErrStale if data is missing or expired.
func (m *Manager) GetPosition() (types.AircraftPosition, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.isStale(GroupPosition) {
		return types.AircraftPosition{}, ErrStale
	}
	return m.position, nil
}

// UpdateInstruments stores new flight instruments data.
func (m *Manager) UpdateInstruments(inst types.FlightInstruments) { //nolint:gocritic
	m.mu.Lock()
	defer m.mu.Unlock()
	m.instruments = inst
	m.lastUpdated[GroupInstruments] = time.Now()
}

// GetInstruments returns the cached instruments, or ErrStale if data is missing or expired.
func (m *Manager) GetInstruments() (types.FlightInstruments, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.isStale(GroupInstruments) {
		return types.FlightInstruments{}, ErrStale
	}
	return m.instruments, nil
}

// UpdateEngine stores new engine data.
func (m *Manager) UpdateEngine(eng types.EngineData) { //nolint:gocritic
	m.mu.Lock()
	defer m.mu.Unlock()
	m.engine = eng
	m.lastUpdated[GroupEngine] = time.Now()
}

// GetEngine returns the cached engine data, or ErrStale if data is missing or expired.
func (m *Manager) GetEngine() (types.EngineData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.isStale(GroupEngine) {
		return types.EngineData{}, ErrStale
	}
	return m.engine, nil
}

// UpdateEnvironment stores new environment data.
func (m *Manager) UpdateEnvironment(env types.Environment) { //nolint:gocritic
	m.mu.Lock()
	defer m.mu.Unlock()
	m.environment = env
	m.lastUpdated[GroupEnvironment] = time.Now()
}

// GetEnvironment returns the cached environment, or ErrStale if data is missing or expired.
func (m *Manager) GetEnvironment() (types.Environment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.isStale(GroupEnvironment) {
		return types.Environment{}, ErrStale
	}
	return m.environment, nil
}

// UpdateAutopilot stores new autopilot state.
func (m *Manager) UpdateAutopilot(ap types.AutopilotState) { //nolint:gocritic
	m.mu.Lock()
	defer m.mu.Unlock()
	m.autopilot = ap
	m.lastUpdated[GroupAutopilot] = time.Now()
}

// GetAutopilot returns the cached autopilot state, or ErrStale if data is missing or expired.
func (m *Manager) GetAutopilot() (types.AutopilotState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.isStale(GroupAutopilot) {
		return types.AutopilotState{}, ErrStale
	}
	return m.autopilot, nil
}

// LastUpdated returns the most recent update time across all groups, or zero if never updated.
func (m *Manager) LastUpdated() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var latest time.Time
	for _, t := range m.lastUpdated {
		if t.After(latest) {
			latest = t
		}
	}
	return latest
}
