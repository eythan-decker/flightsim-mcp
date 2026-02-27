package simconnect

import (
	"context"
	"encoding/binary"
	"io"
	"math"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func defaultTestConfig() Config {
	return Config{
		Host:    "127.0.0.1",
		Port:    4500,
		Timeout: 5 * time.Second,
		AppName: "test-app",
	}
}

// drainOneMessage reads a complete outgoing SimConnect message (16-byte send header + payload)
// from the server side of a net.Pipe.
func drainOneMessage(conn net.Conn) (SendHeader, []byte, error) {
	headerBuf := make([]byte, SendHeaderSize)
	if _, err := io.ReadFull(conn, headerBuf); err != nil {
		return SendHeader{}, nil, err
	}
	h := SendHeader{
		Size:    binary.LittleEndian.Uint32(headerBuf[0:4]),
		Version: binary.LittleEndian.Uint32(headerBuf[4:8]),
		Type:    binary.LittleEndian.Uint32(headerBuf[8:12]),
		ID:      binary.LittleEndian.Uint32(headerBuf[12:16]),
	}
	payloadSize := h.Size - SendHeaderSize
	if payloadSize == 0 {
		return h, nil, nil
	}
	payload := make([]byte, payloadSize)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return SendHeader{}, nil, err
	}
	return h, payload, nil
}

// writeRecvMessage writes a 12-byte receive header + payload to simulate a SimConnect server response.
func writeRecvMessage(conn net.Conn, msgType uint32, payload []byte) error {
	header := make([]byte, RecvHeaderSize)
	binary.LittleEndian.PutUint32(header[0:4], uint32(RecvHeaderSize)+uint32(len(payload))) // #nosec G115
	binary.LittleEndian.PutUint32(header[4:8], 0x06)                                        // server protocol version
	binary.LittleEndian.PutUint32(header[8:12], msgType)
	if _, err := conn.Write(header); err != nil {
		return err
	}
	if len(payload) > 0 {
		if _, err := conn.Write(payload); err != nil {
			return err
		}
	}
	return nil
}

// connectAndDrainOpen connects the client and drains the OPEN message from the server side.
func connectAndDrainOpen(t *testing.T, c *Client) (clientConn, serverConn net.Conn) {
	t.Helper()
	clientConn, serverConn = net.Pipe()

	openDrained := make(chan struct{})
	go func() {
		defer close(openDrained)
		_, _, _ = drainOneMessage(serverConn)
	}()

	err := c.connectWithConn(context.Background(), clientConn)
	require.NoError(t, err)
	<-openDrained
	return clientConn, serverConn
}

func TestNewClient(t *testing.T) {
	cfg := defaultTestConfig()
	c := NewClient(cfg)
	assert.Equal(t, StateDisconnected, c.State())
	assert.Equal(t, cfg.AppName, c.config.AppName)
}

func TestConnectSendsOpenMessage(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	cfg := defaultTestConfig()
	c := NewClient(cfg)

	done := make(chan struct{})
	var receivedHeader SendHeader
	var receivedPayload []byte
	go func() {
		defer close(done)
		receivedHeader, receivedPayload, _ = drainOneMessage(serverConn)
	}()

	err := c.connectWithConn(context.Background(), clientConn)
	require.NoError(t, err)
	clientConn.Close()
	<-done

	// Type should have mask applied
	assert.Equal(t, SendOpen|SendTypeMask, receivedHeader.Type)
	assert.Equal(t, ProtocolVersion, receivedHeader.Version)
	assert.Equal(t, StateConnected, c.State())

	// Payload should be 280 bytes
	require.Len(t, receivedPayload, 280)

	// 256-byte padded app name
	assert.Equal(t, cfg.AppName, string(receivedPayload[0:len(cfg.AppName)]))
	assert.Equal(t, byte(0), receivedPayload[len(cfg.AppName)]) // null-padded

	// Alias "HK" at offset 261
	assert.Equal(t, byte('H'), receivedPayload[261])
	assert.Equal(t, byte('K'), receivedPayload[262])

	// Version numbers at offset 264
	assert.Equal(t, KHMajor, binary.LittleEndian.Uint32(receivedPayload[264:268]))
	assert.Equal(t, KHMinor, binary.LittleEndian.Uint32(receivedPayload[268:272]))
	assert.Equal(t, KHBuildMajor, binary.LittleEndian.Uint32(receivedPayload[272:276]))
	assert.Equal(t, KHBuildMinor, binary.LittleEndian.Uint32(receivedPayload[276:280]))
}

func TestConnectStateTransitions(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()

	go func() {
		buf := make([]byte, 4096)
		for {
			if _, err := serverConn.Read(buf); err != nil {
				return
			}
		}
	}()

	c := NewClient(defaultTestConfig())
	assert.Equal(t, StateDisconnected, c.State())

	err := c.connectWithConn(context.Background(), clientConn)
	require.NoError(t, err)
	assert.Equal(t, StateConnected, c.State())
}

func TestConnectWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := NewClient(defaultTestConfig())
	err := c.connectWithConn(ctx, nil)
	assert.Error(t, err)
	assert.Equal(t, StateDisconnected, c.State())
}

func TestCloseSendsCloseMessage(t *testing.T) {
	c := NewClient(defaultTestConfig())
	_, serverConn := connectAndDrainOpen(t, c)

	done := make(chan SendHeader, 1)
	go func() {
		h, _, err := drainOneMessage(serverConn)
		if err == nil {
			done <- h
		}
	}()

	err := c.Close()
	require.NoError(t, err)
	assert.Equal(t, StateDisconnected, c.State())

	select {
	case h := <-done:
		assert.Equal(t, SendClose|SendTypeMask, h.Type)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for CLOSE message")
	}
}

func TestDoubleCloseIsSafe(t *testing.T) {
	c := NewClient(defaultTestConfig())
	_, serverConn := connectAndDrainOpen(t, c)

	go func() {
		buf := make([]byte, 4096)
		for {
			if _, err := serverConn.Read(buf); err != nil {
				return
			}
		}
	}()

	err := c.Close()
	assert.NoError(t, err)

	err = c.Close()
	assert.NoError(t, err)
}

func TestSendMessageWritesCorrectBytes(t *testing.T) {
	c := NewClient(defaultTestConfig())
	_, serverConn := connectAndDrainOpen(t, c)

	payload := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	done := make(chan struct {
		h       SendHeader
		payload []byte
	}, 1)
	go func() {
		h, p, err := drainOneMessage(serverConn)
		if err == nil {
			done <- struct {
				h       SendHeader
				payload []byte
			}{h, p}
		}
	}()

	err := c.sendMessage(0x99, payload)
	require.NoError(t, err)

	select {
	case msg := <-done:
		assert.Equal(t, uint32(0x99)|SendTypeMask, msg.h.Type)
		assert.Equal(t, uint32(SendHeaderSize+4), msg.h.Size)
		assert.Equal(t, payload, msg.payload)
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestReadMessage(t *testing.T) {
	c := NewClient(defaultTestConfig())
	_, serverConn := connectAndDrainOpen(t, c)

	payload := make([]byte, 8)
	binary.LittleEndian.PutUint64(payload, 0xCAFEBABEDEADBEEF)

	go func() {
		_ = writeRecvMessage(serverConn, RecvSimObjectData, payload)
	}()

	h, data, err := c.readMessage()
	require.NoError(t, err)
	assert.Equal(t, RecvSimObjectData, h.Type)
	assert.Equal(t, payload, data)
}

func TestReadNext(t *testing.T) {
	c := NewClient(defaultTestConfig())
	_, serverConn := connectAndDrainOpen(t, c)

	payload := make([]byte, 8)
	binary.LittleEndian.PutUint64(payload, 0xCAFEBABEDEADBEEF)

	go func() {
		_ = writeRecvMessage(serverConn, RecvSimObjectData, payload)
	}()

	h, data, err := c.ReadNext()
	require.NoError(t, err)
	assert.Equal(t, RecvSimObjectData, h.Type)
	assert.Equal(t, payload, data)
}

func TestReadNextWhenNotConnected(t *testing.T) {
	c := NewClient(defaultTestConfig())
	_, _, err := c.ReadNext()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotConnected)
}

func TestAddToDataDefinition(t *testing.T) {
	c := NewClient(defaultTestConfig())
	_, serverConn := connectAndDrainOpen(t, c)

	done := make(chan struct {
		h       SendHeader
		payload []byte
	}, 1)
	go func() {
		h, p, err := drainOneMessage(serverConn)
		if err == nil {
			done <- struct {
				h       SendHeader
				payload []byte
			}{h, p}
		}
	}()

	err := c.AddToDataDefinition(1, PlaneLatitude)
	require.NoError(t, err)

	select {
	case msg := <-done:
		assert.Equal(t, SendAddToDataDef|SendTypeMask, msg.h.Type)

		p := msg.payload
		require.Len(t, p, 528)

		// defID
		defID := binary.LittleEndian.Uint32(p[0:4])
		assert.Equal(t, uint32(1), defID)

		// 256-byte padded datum name
		assert.Equal(t, "PLANE LATITUDE", string(p[4:4+len("PLANE LATITUDE")]))
		assert.Equal(t, byte(0), p[4+len("PLANE LATITUDE")])

		// 256-byte padded units name at offset 260
		assert.Equal(t, "degrees", string(p[260:260+len("degrees")]))
		assert.Equal(t, byte(0), p[260+len("degrees")])

		// dataType at offset 516 = 4 (FLOAT64)
		dt := binary.LittleEndian.Uint32(p[516:520])
		assert.Equal(t, uint32(DataTypeFloat64), dt)

		// epsilon at offset 520 = 0.0
		eps := math.Float32frombits(binary.LittleEndian.Uint32(p[520:524]))
		assert.Equal(t, float32(0.0), eps)

		// datumId at offset 524 = 0xffffffff
		datumID := binary.LittleEndian.Uint32(p[524:528])
		assert.Equal(t, uint32(0xffffffff), datumID)

	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestRequestData(t *testing.T) {
	c := NewClient(defaultTestConfig())
	_, serverConn := connectAndDrainOpen(t, c)

	done := make(chan struct {
		h       SendHeader
		payload []byte
	}, 1)
	go func() {
		h, p, err := drainOneMessage(serverConn)
		if err == nil {
			done <- struct {
				h       SendHeader
				payload []byte
			}{h, p}
		}
	}()

	err := c.RequestData(1, 0, 100)
	require.NoError(t, err)

	select {
	case msg := <-done:
		assert.Equal(t, SendRequestData|SendTypeMask, msg.h.Type)

		p := msg.payload
		require.Len(t, p, 32)

		requestID := binary.LittleEndian.Uint32(p[0:4])
		defID := binary.LittleEndian.Uint32(p[4:8])
		objectID := binary.LittleEndian.Uint32(p[8:12])
		period := binary.LittleEndian.Uint32(p[12:16])
		flags := binary.LittleEndian.Uint32(p[16:20])
		origin := binary.LittleEndian.Uint32(p[20:24])
		interval := binary.LittleEndian.Uint32(p[24:28])
		limit := binary.LittleEndian.Uint32(p[28:32])

		assert.Equal(t, uint32(100), requestID)
		assert.Equal(t, uint32(1), defID)
		assert.Equal(t, uint32(0), objectID)
		assert.Equal(t, uint32(1), period, "should be PERIOD_ONCE")
		assert.Equal(t, uint32(0), flags)
		assert.Equal(t, uint32(0), origin)
		assert.Equal(t, uint32(0), interval)
		assert.Equal(t, uint32(0), limit)

	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}
