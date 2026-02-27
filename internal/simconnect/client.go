package simconnect

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
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

	// Build KittyHawk OPEN payload (280 bytes):
	//   256-byte zero-padded app name
	//   int32(0) reserved
	//   byte(0x00)
	//   3-byte alias "HK\x00"
	//   int32 major, minor, buildMajor, buildMinor
	payload := make([]byte, 0, 280)

	appName := make([]byte, 256)
	copy(appName, c.config.AppName)
	payload = append(payload, appName...)

	payload = binary.LittleEndian.AppendUint32(payload, 0) // reserved
	payload = append(payload, 0x00)                        // separator

	alias := make([]byte, 3)
	copy(alias, KHAlias)
	payload = append(payload, alias...)

	payload = binary.LittleEndian.AppendUint32(payload, KHMajor)
	payload = binary.LittleEndian.AppendUint32(payload, KHMinor)
	payload = binary.LittleEndian.AppendUint32(payload, KHBuildMajor)
	payload = binary.LittleEndian.AppendUint32(payload, KHBuildMinor)

	if err := c.sendMessageLocked(SendOpen, payload); err != nil {
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
	_ = c.sendMessageLocked(SendClose, nil)

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
	header := EncodeSendHeader(msgType, id, len(payload))

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
func (c *Client) ReadNext() (RecvHeader, []byte, error) {
	return c.readMessage()
}

// readMessage reads a complete framed message from the connection.
func (c *Client) readMessage() (RecvHeader, []byte, error) {
	if c.conn == nil {
		return RecvHeader{}, nil, ErrNotConnected
	}

	headerBuf := make([]byte, RecvHeaderSize)
	if _, err := io.ReadFull(c.conn, headerBuf); err != nil {
		return RecvHeader{}, nil, fmt.Errorf("read header: %w", err)
	}

	h, err := DecodeRecvHeader(headerBuf)
	if err != nil {
		return RecvHeader{}, nil, err
	}

	payloadSize := h.Size - RecvHeaderSize
	if payloadSize == 0 {
		return h, nil, nil
	}

	payload := make([]byte, payloadSize)
	if _, err := io.ReadFull(c.conn, payload); err != nil {
		return RecvHeader{}, nil, fmt.Errorf("read payload: %w", err)
	}
	return h, payload, nil
}

// AddToDataDefinition sends an ADD_TO_DATA_DEFINITION message to register
// a SimVar with the given definition ID.
func (c *Client) AddToDataDefinition(defID uint32, simvar SimVarDef) error {
	// KittyHawk payload layout (528 bytes):
	//   int32:      defID
	//   char[256]:  datum name (zero-padded)
	//   char[256]:  units name (zero-padded)
	//   int32:      dataType (4 for FLOAT64)
	//   float32:    epsilon (0.0)
	//   int32:      datumId (0xffffffff = UNUSED)
	payload := make([]byte, 0, 528)

	payload = binary.LittleEndian.AppendUint32(payload, defID)

	datumName := make([]byte, 256)
	copy(datumName, simvar.Name)
	payload = append(payload, datumName...)

	unitsName := make([]byte, 256)
	copy(unitsName, simvar.Unit)
	payload = append(payload, unitsName...)

	payload = binary.LittleEndian.AppendUint32(payload, uint32(simvar.DataType)) // #nosec G115 -- DataType is a small enum value
	payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(0.0))   // epsilon
	payload = binary.LittleEndian.AppendUint32(payload, 0xffffffff)              // datumId UNUSED

	return c.sendMessage(SendAddToDataDef, payload)
}

// RequestData sends a REQUEST_DATA message to start receiving data for
// the given definition ID, object ID, and request ID.
func (c *Client) RequestData(defID, objectID, requestID uint32) error {
	// KittyHawk payload layout (32 bytes):
	//   int32: requestID, defID, objectID
	//   int32: period(1=PERIOD_ONCE), flags(0), origin(0), interval(0), limit(0)
	payload := make([]byte, 0, 32)
	payload = binary.LittleEndian.AppendUint32(payload, requestID)
	payload = binary.LittleEndian.AppendUint32(payload, defID)
	payload = binary.LittleEndian.AppendUint32(payload, objectID)
	payload = binary.LittleEndian.AppendUint32(payload, 1) // PERIOD_ONCE
	payload = binary.LittleEndian.AppendUint32(payload, 0) // flags
	payload = binary.LittleEndian.AppendUint32(payload, 0) // origin
	payload = binary.LittleEndian.AppendUint32(payload, 0) // interval
	payload = binary.LittleEndian.AppendUint32(payload, 0) // limit

	return c.sendMessage(SendRequestData, payload)
}
