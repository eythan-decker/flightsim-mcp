package simconnect

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Config holds SimConnect client configuration.
type Config struct {
	Host    string
	Port    int
	Timeout time.Duration
	AppName string
}

// ConnectionState represents the client's connection lifecycle.
type ConnectionState int32

const (
	StateDisconnected ConnectionState = iota
	StateConnecting
	StateConnected
	StateReconnecting
)

// Client manages a TCP connection to SimConnect.
type Client struct {
	config Config
	conn   net.Conn
	state  atomic.Int32
	mu     sync.Mutex
	nextID atomic.Uint32
}

// NewClient creates a new SimConnect client.
func NewClient(cfg Config) *Client {
	c := &Client{config: cfg}
	c.state.Store(int32(StateDisconnected))
	return c
}

// State returns the current connection state.
func (c *Client) State() ConnectionState {
	return ConnectionState(c.state.Load())
}

// Connect establishes a TCP connection and sends the OPEN message.
func (c *Client) Connect(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	dialer := net.Dialer{Timeout: c.config.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("simconnect dial: %w", err)
	}
	return c.connectWithConn(ctx, conn)
}

// connectWithConn performs the connection handshake on an existing net.Conn.
// This is separated from Connect to allow testing with net.Pipe().
func (c *Client) connectWithConn(ctx context.Context, conn net.Conn) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("simconnect connect: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.state.Store(int32(StateConnecting))
	c.conn = conn

	// Build OPEN payload: null-terminated app name
	appNameBytes := append([]byte(c.config.AppName), 0)
	if err := c.sendMessageLocked(MsgOpen, appNameBytes); err != nil {
		c.state.Store(int32(StateDisconnected))
		return fmt.Errorf("simconnect open: %w", err)
	}

	c.state.Store(int32(StateConnected))
	return nil
}

// Close sends a CLOSE message and shuts down the TCP connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil
	}

	// Best-effort CLOSE message — ignore write errors on already-closed conn
	_ = c.sendMessageLocked(MsgClose, nil)

	err := c.conn.Close()
	c.conn = nil
	c.state.Store(int32(StateDisconnected))
	return err
}

// sendMessage sends a framed message (header + payload) over the connection.
// Thread-safe: acquires the mutex.
func (c *Client) sendMessage(msgType uint32, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sendMessageLocked(msgType, payload)
}

// sendMessageLocked sends a message; caller must hold c.mu.
func (c *Client) sendMessageLocked(msgType uint32, payload []byte) error {
	if c.conn == nil {
		return ErrNotConnected
	}

	id := c.nextID.Add(1)
	header := EncodeHeader(msgType, id, len(payload))

	if _, err := c.conn.Write(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	if len(payload) > 0 {
		if _, err := c.conn.Write(payload); err != nil {
			return fmt.Errorf("write payload: %w", err)
		}
	}
	return nil
}

// ReadNext reads the next complete framed message from the SimConnect connection.
func (c *Client) ReadNext() (Header, []byte, error) {
	return c.readMessage()
}

// readMessage reads a complete framed message from the connection.
func (c *Client) readMessage() (Header, []byte, error) {
	if c.conn == nil {
		return Header{}, nil, ErrNotConnected
	}

	headerBuf := make([]byte, HeaderSize)
	if _, err := io.ReadFull(c.conn, headerBuf); err != nil {
		return Header{}, nil, fmt.Errorf("read header: %w", err)
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
	if _, err := io.ReadFull(c.conn, payload); err != nil {
		return Header{}, nil, fmt.Errorf("read payload: %w", err)
	}
	return h, payload, nil
}

// AddToDataDefinition sends an ADD_TO_DATA_DEFINITION message to register
// a SimVar with the given definition ID.
func (c *Client) AddToDataDefinition(defID uint32, simvar SimVarDef) error {
	// Payload layout:
	//   defID        uint32 (4 bytes)
	//   varName      null-terminated string (variable length)
	//   unitName     null-terminated string (variable length)
	//   dataType     uint32 (4 bytes)
	varName := append([]byte(simvar.Name), 0)
	unitName := append([]byte(simvar.Unit), 0)

	payload := make([]byte, 0, 4+len(varName)+len(unitName)+4)
	payload = binary.LittleEndian.AppendUint32(payload, defID)
	payload = append(payload, varName...)
	payload = append(payload, unitName...)
	payload = binary.LittleEndian.AppendUint32(payload, uint32(simvar.DataType)) //nolint:gosec // DataType is a small enum value, conversion is safe

	return c.sendMessage(MsgAddToDataDef, payload)
}

// RequestData sends a REQUEST_DATA message to start receiving data for
// the given definition ID, object ID, and request ID.
func (c *Client) RequestData(defID, objectID, requestID uint32) error {
	// Payload layout:
	//   requestID    uint32 (4 bytes)
	//   defID        uint32 (4 bytes)
	//   objectID     uint32 (4 bytes)
	payload := make([]byte, 0, 12)
	payload = binary.LittleEndian.AppendUint32(payload, requestID)
	payload = binary.LittleEndian.AppendUint32(payload, defID)
	payload = binary.LittleEndian.AppendUint32(payload, objectID)

	return c.sendMessage(MsgRequestData, payload)
}
