package simconnect

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeHeader(t *testing.T) {
	tests := []struct {
		name        string
		msgType     uint32
		msgID       uint32
		payloadSize int
		wantSize    uint32
	}{
		{
			name:        "zero payload",
			msgType:     MsgOpen,
			msgID:       1,
			payloadSize: 0,
			wantSize:    HeaderSize,
		},
		{
			name:        "with payload",
			msgType:     MsgRequestData,
			msgID:       42,
			payloadSize: 100,
			wantSize:    HeaderSize + 100,
		},
		{
			name:        "large payload",
			msgType:     MsgSetDataDefinition,
			msgID:       999,
			payloadSize: 65536,
			wantSize:    HeaderSize + 65536,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := EncodeHeader(tt.msgType, tt.msgID, tt.payloadSize)
			require.Len(t, data, int(HeaderSize))

			// Verify little-endian encoding
			size := binary.LittleEndian.Uint32(data[0:4])
			version := binary.LittleEndian.Uint32(data[4:8])
			msgType := binary.LittleEndian.Uint32(data[8:12])
			msgID := binary.LittleEndian.Uint32(data[12:16])

			assert.Equal(t, tt.wantSize, size)
			assert.Equal(t, uint32(ProtocolVersion), version)
			assert.Equal(t, tt.msgType, msgType)
			assert.Equal(t, tt.msgID, msgID)
		})
	}
}

func TestDecodeHeader(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    Header
		wantErr bool
	}{
		{
			name: "valid header",
			data: func() []byte {
				b := make([]byte, HeaderSize)
				binary.LittleEndian.PutUint32(b[0:4], 24)
				binary.LittleEndian.PutUint32(b[4:8], ProtocolVersion)
				binary.LittleEndian.PutUint32(b[8:12], MsgOpen)
				binary.LittleEndian.PutUint32(b[12:16], 1)
				return b
			}(),
			want: Header{Size: 24, Version: ProtocolVersion, Type: MsgOpen, ID: 1},
		},
		{
			name:    "too short",
			data:    make([]byte, 8),
			wantErr: true,
		},
		{
			name:    "nil data",
			data:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeHeader(tt.data)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	tests := []struct {
		name        string
		msgType     uint32
		msgID       uint32
		payloadSize int
	}{
		{"open message", MsgOpen, 0, 256},
		{"close message", MsgClose, 1, 0},
		{"request data", MsgRequestData, 100, 48},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := EncodeHeader(tt.msgType, tt.msgID, tt.payloadSize)
			decoded, err := DecodeHeader(encoded)
			require.NoError(t, err)

			assert.Equal(t, uint32(HeaderSize)+uint32(tt.payloadSize), decoded.Size)
			assert.Equal(t, uint32(ProtocolVersion), decoded.Version)
			assert.Equal(t, tt.msgType, decoded.Type)
			assert.Equal(t, tt.msgID, decoded.ID)
		})
	}
}

func TestLittleEndianByteOrder(t *testing.T) {
	// Verify specific byte layout for MsgOpen with ID=1, payloadSize=0
	data := EncodeHeader(MsgOpen, 1, 0)

	// Size = 16 (0x10) in little-endian: 10 00 00 00
	assert.Equal(t, byte(0x10), data[0])
	assert.Equal(t, byte(0x00), data[1])
	assert.Equal(t, byte(0x00), data[2])
	assert.Equal(t, byte(0x00), data[3])

	// Version = 4 in little-endian: 04 00 00 00
	assert.Equal(t, byte(0x04), data[4])
	assert.Equal(t, byte(0x00), data[5])

	// Type = MsgOpen (0x0001): 01 00 00 00
	assert.Equal(t, byte(0x01), data[8])
	assert.Equal(t, byte(0x00), data[9])

	// ID = 1: 01 00 00 00
	assert.Equal(t, byte(0x01), data[12])
	assert.Equal(t, byte(0x00), data[13])
}
