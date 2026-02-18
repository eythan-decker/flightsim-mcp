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

// mockUpdater captures Update calls for assertion.
type mockUpdater struct {
	mu    sync.Mutex
	calls []types.AircraftPosition
}

func (m *mockUpdater) Update(pos types.AircraftPosition) { //nolint:gocritic
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, pos)
}

func (m *mockUpdater) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func (m *mockUpdater) LastCall() (types.AircraftPosition, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.calls) == 0 {
		return types.AircraftPosition{}, false
	}
	return m.calls[len(m.calls)-1], true
}

// buildPositionPayload builds a 96-byte position payload with the given values.
func buildPositionPayload(vals [12]float64) []byte { //nolint:gocritic
	buf := make([]byte, 96)
	for i, v := range vals {
		binary.LittleEndian.PutUint64(buf[i*8:], math.Float64bits(v))
	}
	return buf
}

func newConnectedPoller(t *testing.T, updater PositionUpdater, cfg PollerConfig) (*Poller, net.Conn) {
	t.Helper()
	c := NewClient(defaultTestConfig())
	_, serverConn := connectAndDrainOpen(t, c)
	p := NewPoller(c, updater, cfg)
	return p, serverConn
}

func TestRegisterSimVarsSendsAllDefinitions(t *testing.T) {
	updater := &mockUpdater{}
	p, serverConn := newConnectedPoller(t, updater, DefaultPollerConfig())

	received := make(chan Header, len(PositionSimVars))
	go func() {
		for i := 0; i < len(PositionSimVars); i++ {
			h, _, err := drainOneMessage(serverConn)
			if err != nil {
				return
			}
			received <- h
		}
	}()

	err := p.RegisterSimVars()
	require.NoError(t, err)

	for i := 0; i < len(PositionSimVars); i++ {
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
		// Drain the first RequestData that Start sends immediately on first tick
		// (we use a long interval so it won't tick again, but Start drains the ticker channel)
		// Send a SimObjectData response.
		header := EncodeHeader(MsgSimObjectData, ReqIDPosition, len(payload))
		_, _ = serverConn.Write(header)
		_, _ = serverConn.Write(payload)
		// Keep the connection open so readLoop doesn't exit
		<-ctx.Done()
	}()

	go func() { _ = p.Start(ctx) }()

	require.Eventually(t, func() bool {
		return updater.CallCount() > 0
	}, 2*time.Second, 10*time.Millisecond, "expected at least one Update call")

	pos, ok := updater.LastCall()
	require.True(t, ok)
	assert.InDelta(t, 47.6062, pos.Latitude, 1e-9)
	assert.InDelta(t, -122.3321, pos.Longitude, 1e-9)
	assert.InDelta(t, 35000.0, pos.AltitudeMSL, 1e-9)
}

func TestReadLoopExitsOnEOF(t *testing.T) {
	updater := &mockUpdater{}
	cfg := PollerConfig{PollInterval: 10 * time.Second}
	p, serverConn := newConnectedPoller(t, updater, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Close the server side to cause EOF on the client
	go func() {
		time.Sleep(30 * time.Millisecond)
		serverConn.Close()
	}()

	err := p.Start(ctx)
	assert.Error(t, err) // EOF or broken pipe
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
