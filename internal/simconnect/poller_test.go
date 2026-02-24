package simconnect

import (
	"context"
	"encoding/binary"
	"math"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eytandecker/flightsim-mcp/pkg/types"
)

// mockUpdater captures all Update calls for assertion.
type mockUpdater struct {
	mu           sync.Mutex
	positions    []types.AircraftPosition
	instruments  []types.FlightInstruments
	engines      []types.EngineData
	environments []types.Environment
	autopilots   []types.AutopilotState
}

func (m *mockUpdater) Update(pos types.AircraftPosition) { //nolint:gocritic
	m.mu.Lock()
	defer m.mu.Unlock()
	m.positions = append(m.positions, pos)
}

func (m *mockUpdater) UpdateInstruments(inst types.FlightInstruments) { //nolint:gocritic
	m.mu.Lock()
	defer m.mu.Unlock()
	m.instruments = append(m.instruments, inst)
}

func (m *mockUpdater) UpdateEngine(eng types.EngineData) { //nolint:gocritic
	m.mu.Lock()
	defer m.mu.Unlock()
	m.engines = append(m.engines, eng)
}

func (m *mockUpdater) UpdateEnvironment(env types.Environment) { //nolint:gocritic
	m.mu.Lock()
	defer m.mu.Unlock()
	m.environments = append(m.environments, env)
}

func (m *mockUpdater) UpdateAutopilot(ap types.AutopilotState) { //nolint:gocritic
	m.mu.Lock()
	defer m.mu.Unlock()
	m.autopilots = append(m.autopilots, ap)
}

func (m *mockUpdater) PositionCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.positions)
}

func (m *mockUpdater) LastPosition() (types.AircraftPosition, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.positions) == 0 {
		return types.AircraftPosition{}, false
	}
	return m.positions[len(m.positions)-1], true
}

func (m *mockUpdater) InstrumentsCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.instruments)
}

func (m *mockUpdater) EngineCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.engines)
}

func (m *mockUpdater) EnvironmentCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.environments)
}

func (m *mockUpdater) AutopilotCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.autopilots)
}

// buildFloat64Payload builds a payload of N float64 values.
func buildFloat64Payload(vals []float64) []byte {
	buf := make([]byte, len(vals)*8)
	for i, v := range vals {
		binary.LittleEndian.PutUint64(buf[i*8:], math.Float64bits(v))
	}
	return buf
}

// buildPositionPayload builds a 96-byte position payload with the given values.
func buildPositionPayload(vals [12]float64) []byte { //nolint:gocritic
	buf := make([]byte, 96)
	for i, v := range vals {
		binary.LittleEndian.PutUint64(buf[i*8:], math.Float64bits(v))
	}
	return buf
}

func newConnectedPoller(t *testing.T, updater StateUpdater, cfg PollerConfig) (*Poller, net.Conn) {
	t.Helper()
	c := NewClient(defaultTestConfig())
	_, serverConn := connectAndDrainOpen(t, c)
	p := NewPoller(c, updater, cfg)
	return p, serverConn
}

func TestRegisterSimVarsSendsAllDefinitions(t *testing.T) {
	updater := &mockUpdater{}
	p, serverConn := newConnectedPoller(t, updater, DefaultPollerConfig())

	// Total SimVars across all 5 groups: 12 + 11 + 20 + 8 + 12 = 63
	totalVars := len(PositionSimVars) + len(InstrumentsSimVars) + len(EngineSimVars) +
		len(EnvironmentSimVars) + len(AutopilotSimVars)
	assert.Equal(t, 63, totalVars)

	received := make(chan Header, totalVars)
	go func() {
		for i := 0; i < totalVars; i++ {
			h, _, err := drainOneMessage(serverConn)
			if err != nil {
				return
			}
			received <- h
		}
	}()

	err := p.RegisterSimVars()
	require.NoError(t, err)

	for i := 0; i < totalVars; i++ {
		select {
		case h := <-received:
			assert.Equal(t, uint32(MsgAddToDataDef), h.Type, "message %d", i)
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for AddToDataDef message %d", i)
		}
	}
}

func TestStartSendsRequestDataOnTick(t *testing.T) {
	updater := &mockUpdater{}
	cfg := PollerConfig{PollInterval: 20 * time.Millisecond}
	p, serverConn := newConnectedPoller(t, updater, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	requestSeen := make(chan struct{}, 1)
	go func() {
		for {
			h, _, err := drainOneMessage(serverConn)
			if err != nil {
				return
			}
			if h.Type == uint32(MsgRequestData) {
				select {
				case requestSeen <- struct{}{}:
				default:
				}
			}
		}
	}()

	go func() { _ = p.Start(ctx) }()

	select {
	case <-requestSeen:
		// pass
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for RequestData message")
	}
}

func TestReadLoopProcessesSimObjectData(t *testing.T) {
	updater := &mockUpdater{}
	cfg := PollerConfig{PollInterval: 10 * time.Second} // long interval — we trigger manually
	p, serverConn := newConnectedPoller(t, updater, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	wantVals := [12]float64{
		47.6062, -122.3321, 35000.0, 34950.0,
		270.0, 268.5, 450.0, 455.0, 448.0, 500.0, 2.5, -1.0,
	}
	payload := buildPositionPayload(wantVals)

	go func() {
		header := EncodeHeader(MsgSimObjectData, ReqIDPosition, len(payload))
		_, _ = serverConn.Write(header)
		_, _ = serverConn.Write(payload)
		<-ctx.Done()
	}()

	go func() { _ = p.Start(ctx) }()

	require.Eventually(t, func() bool {
		return updater.PositionCount() > 0
	}, 2*time.Second, 10*time.Millisecond, "expected at least one Update call")

	pos, ok := updater.LastPosition()
	require.True(t, ok)
	assert.InDelta(t, 47.6062, pos.Latitude, 1e-9)
	assert.InDelta(t, -122.3321, pos.Longitude, 1e-9)
	assert.InDelta(t, 35000.0, pos.AltitudeMSL, 1e-9)
}

func TestReadLoopDispatchesInstruments(t *testing.T) {
	updater := &mockUpdater{}
	cfg := PollerConfig{PollInterval: 10 * time.Second}
	p, serverConn := newConnectedPoller(t, updater, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	vals := make([]float64, 11)
	vals[0] = 35000.0 // IndicatedAltitude
	vals[5] = 0.78    // AirspeedMach
	payload := buildFloat64Payload(vals)

	go func() {
		header := EncodeHeader(MsgSimObjectData, ReqIDInstruments, len(payload))
		_, _ = serverConn.Write(header)
		_, _ = serverConn.Write(payload)
		<-ctx.Done()
	}()

	go func() { _ = p.Start(ctx) }()

	require.Eventually(t, func() bool {
		return updater.InstrumentsCount() > 0
	}, 2*time.Second, 10*time.Millisecond)
}

func TestReadLoopDispatchesEngine(t *testing.T) {
	updater := &mockUpdater{}
	cfg := PollerConfig{PollInterval: 10 * time.Second}
	p, serverConn := newConnectedPoller(t, updater, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	vals := make([]float64, 20)
	vals[0] = 2.0 // NumberOfEngines
	payload := buildFloat64Payload(vals)

	go func() {
		header := EncodeHeader(MsgSimObjectData, ReqIDEngine, len(payload))
		_, _ = serverConn.Write(header)
		_, _ = serverConn.Write(payload)
		<-ctx.Done()
	}()

	go func() { _ = p.Start(ctx) }()

	require.Eventually(t, func() bool {
		return updater.EngineCount() > 0
	}, 2*time.Second, 10*time.Millisecond)
}

func TestReadLoopDispatchesEnvironment(t *testing.T) {
	updater := &mockUpdater{}
	cfg := PollerConfig{PollInterval: 10 * time.Second}
	p, serverConn := newConnectedPoller(t, updater, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	vals := make([]float64, 8)
	vals[0] = 15.0 // WindVelocity
	vals[2] = -5.5 // Temperature
	payload := buildFloat64Payload(vals)

	go func() {
		header := EncodeHeader(MsgSimObjectData, ReqIDEnvironment, len(payload))
		_, _ = serverConn.Write(header)
		_, _ = serverConn.Write(payload)
		<-ctx.Done()
	}()

	go func() { _ = p.Start(ctx) }()

	require.Eventually(t, func() bool {
		return updater.EnvironmentCount() > 0
	}, 2*time.Second, 10*time.Millisecond)
}

func TestReadLoopDispatchesAutopilot(t *testing.T) {
	updater := &mockUpdater{}
	cfg := PollerConfig{PollInterval: 10 * time.Second}
	p, serverConn := newConnectedPoller(t, updater, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	vals := make([]float64, 12)
	vals[0] = 1.0   // Master
	vals[8] = 270.0 // HeadingLockDir
	payload := buildFloat64Payload(vals)

	go func() {
		header := EncodeHeader(MsgSimObjectData, ReqIDAutopilot, len(payload))
		_, _ = serverConn.Write(header)
		_, _ = serverConn.Write(payload)
		<-ctx.Done()
	}()

	go func() { _ = p.Start(ctx) }()

	require.Eventually(t, func() bool {
		return updater.AutopilotCount() > 0
	}, 2*time.Second, 10*time.Millisecond)
}

func TestReadLoopExitsOnEOF(t *testing.T) {
	updater := &mockUpdater{}
	cfg := PollerConfig{PollInterval: 10 * time.Second}
	p, serverConn := newConnectedPoller(t, updater, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		time.Sleep(30 * time.Millisecond)
		serverConn.Close()
	}()

	err := p.Start(ctx)
	assert.Error(t, err)
}

func TestStartExitsWhenContextCancelled(t *testing.T) {
	updater := &mockUpdater{}
	cfg := PollerConfig{PollInterval: 10 * time.Second}
	p, serverConn := newConnectedPoller(t, updater, cfg)
	defer serverConn.Close()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- p.Start(ctx)
	}()

	cancel()

	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("Start did not exit after context cancellation")
	}
}
