package simconnect

import (
	"context"
	"encoding/binary"
	"io"
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

// drainOneMessage reads a complete SimConnect message (header + payload) from conn.
// Returns the header and payload, or closes done on error.
func drainOneMessage(conn net.Conn) (Header, []byte, error) {
	headerBuf := make([]byte, HeaderSize)
	if _, err := io.ReadFull(conn, headerBuf); err != nil {
		return Header{}, nil, err
	}
	h, err := DecodeHeader(headerBuf)
	if err != nil {
		return Header{}, nil, err
	}
	payloadSize := h.Size - HeaderSize
	if payloadSize == 0 {
		return h, nil, nil
	}
	payload := make([]byte, payloadSize)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return Header{}, nil, err
	}
	return h, payload, nil
}

// connectAndDrainOpen connects the client and drains the OPEN message from the server side.
// Returns the client connection and server connection for further use.
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

	// Read what the client sends on connect
	done := make(chan struct{})
	var receivedHeader Header
	var receivedPayload []byte
	go func() {
		defer close(done)
		receivedHeader, receivedPayload, _ = drainOneMessage(serverConn)
	}()

	err := c.connectWithConn(context.Background(), clientConn)
	require.NoError(t, err)

	// Close client side to unblock server read if needed
	clientConn.Close()
	<-done

	assert.Equal(t, uint32(MsgOpen), receivedHeader.Type)
	assert.Equal(t, uint32(ProtocolVersion), receivedHeader.Version)
	assert.Equal(t, StateConnected, c.State())
	assert.Contains(t, string(receivedPayload), cfg.AppName)
}

func TestConnectStateTransitions(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()

	// Drain server side so writes don't block
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

	// Read the CLOSE message
	done := make(chan Header, 1)
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
		assert.Equal(t, uint32(MsgClose), h.Type)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for CLOSE message")
	}
}

func TestDoubleCloseIsSafe(t *testing.T) {
	c := NewClient(defaultTestConfig())
	_, serverConn := connectAndDrainOpen(t, c)

	// Keep draining so CLOSE write doesn't block
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
		h       Header
		payload []byte
	}, 1)
	go func() {
		h, p, err := drainOneMessage(serverConn)
		if err == nil {
			done <- struct {
				h       Header
				payload []byte
			}{h, p}
		}
	}()

	_, err := c.sendMessage(0x9999, payload)
	require.NoError(t, err)

	select {
	case msg := <-done:
		assert.Equal(t, uint32(0x9999), msg.h.Type)
		assert.Equal(t, uint32(HeaderSize+4), msg.h.Size)
		assert.Equal(t, payload, msg.payload)
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestReadMessage(t *testing.T) {
	c := NewClient(defaultTestConfig())
	_, serverConn := connectAndDrainOpen(t, c)

	// Server sends a message
	payload := make([]byte, 8)
	binary.LittleEndian.PutUint64(payload, 0xCAFEBABEDEADBEEF)

	go func() {
		header := EncodeHeader(MsgSimObjectData, 42, len(payload))
		_, _ = serverConn.Write(header)
		_, _ = serverConn.Write(payload)
	}()

	h, data, err := c.readMessage()
	require.NoError(t, err)
	assert.Equal(t, uint32(MsgSimObjectData), h.Type)
	assert.Equal(t, uint32(42), h.ID)
	assert.Equal(t, payload, data)
}

func TestAddToDataDefinition(t *testing.T) {
	c := NewClient(defaultTestConfig())
	_, serverConn := connectAndDrainOpen(t, c)

	done := make(chan struct {
		h       Header
		payload []byte
	}, 1)
	go func() {
		h, p, err := drainOneMessage(serverConn)
		if err == nil {
			done <- struct {
				h       Header
				payload []byte
			}{h, p}
		}
	}()

	err := c.AddToDataDefinition(1, PlaneLatitude)
	require.NoError(t, err)

	select {
	case msg := <-done:
		assert.Equal(t, uint32(MsgAddToDataDef), msg.h.Type)

		// Verify payload layout
		p := msg.payload
		defID := binary.LittleEndian.Uint32(p[0:4])
		assert.Equal(t, uint32(1), defID)

		// Find null-terminated var name after defID
		nameStart := 4
		nameEnd := nameStart
		for nameEnd < len(p) && p[nameEnd] != 0 {
			nameEnd++
		}
		assert.Equal(t, "PLANE LATITUDE", string(p[nameStart:nameEnd]))

		// Find null-terminated unit name after var name + null byte
		unitStart := nameEnd + 1
		unitEnd := unitStart
		for unitEnd < len(p) && p[unitEnd] != 0 {
			unitEnd++
		}
		assert.Equal(t, "degrees", string(p[unitStart:unitEnd]))

		// Data type after unit name + null byte
		dtStart := unitEnd + 1
		dt := binary.LittleEndian.Uint32(p[dtStart : dtStart+4])
		assert.Equal(t, uint32(DataTypeFloat64), dt)

	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestRequestData(t *testing.T) {
	c := NewClient(defaultTestConfig())
	_, serverConn := connectAndDrainOpen(t, c)

	done := make(chan struct {
		h       Header
		payload []byte
	}, 1)
	go func() {
		h, p, err := drainOneMessage(serverConn)
		if err == nil {
			done <- struct {
				h       Header
				payload []byte
			}{h, p}
		}
	}()

	err := c.RequestData(1, 0, 100)
	require.NoError(t, err)

	select {
	case msg := <-done:
		assert.Equal(t, uint32(MsgRequestData), msg.h.Type)

		p := msg.payload
		require.Len(t, p, 12)

		requestID := binary.LittleEndian.Uint32(p[0:4])
		defID := binary.LittleEndian.Uint32(p[4:8])
		objectID := binary.LittleEndian.Uint32(p[8:12])

		assert.Equal(t, uint32(100), requestID)
		assert.Equal(t, uint32(1), defID)
		assert.Equal(t, uint32(0), objectID)

	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}
