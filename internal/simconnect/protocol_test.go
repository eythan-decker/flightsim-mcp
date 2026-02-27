package simconnect

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeSendHeader(t *testing.T) {
	tests := []struct {
		name        string
		msgType     uint32
		msgID       uint32
		payloadSize int
		wantSize    uint32
		wantType    uint32
	}{
		{
			name:        "OPEN with zero payload",
			msgType:     SendOpen,
			msgID:       1,
			payloadSize: 0,
			wantSize:    SendHeaderSize,
			wantType:    SendOpen | SendTypeMask,
		},
		{
			name:        "RequestData with payload",
			msgType:     SendRequestData,
			msgID:       42,
			payloadSize: 32,
			wantSize:    SendHeaderSize + 32,
			wantType:    SendRequestData | SendTypeMask,
		},
		{
			name:        "AddToDataDef with large payload",
			msgType:     SendAddToDataDef,
			msgID:       999,
			payloadSize: 528,
			wantSize:    SendHeaderSize + 528,
			wantType:    SendAddToDataDef | SendTypeMask,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := EncodeSendHeader(tt.msgType, tt.msgID, tt.payloadSize)
			require.Len(t, data, int(SendHeaderSize))

			size := binary.LittleEndian.Uint32(data[0:4])
			version := binary.LittleEndian.Uint32(data[4:8])
			msgType := binary.LittleEndian.Uint32(data[8:12])
			msgID := binary.LittleEndian.Uint32(data[12:16])

			assert.Equal(t, tt.wantSize, size)
			assert.Equal(t, ProtocolVersion, version)
			assert.Equal(t, tt.wantType, msgType, "type should have mask applied")
			assert.Equal(t, tt.msgID, msgID)
		})
	}
}

func TestDecodeRecvHeader(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    RecvHeader
		wantErr bool
	}{
		{
			name: "valid SimObjectData header",
			data: func() []byte {
				b := make([]byte, RecvHeaderSize)
				binary.LittleEndian.PutUint32(b[0:4], 108)  // size
				binary.LittleEndian.PutUint32(b[4:8], 0x06) // server protocol version
				binary.LittleEndian.PutUint32(b[8:12], RecvSimObjectData)
				return b
			}(),
			want: RecvHeader{Size: 108, Version: 0x06, Type: RecvSimObjectData},
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
			got, err := DecodeRecvHeader(tt.data)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLittleEndianByteOrder(t *testing.T) {
	data := EncodeSendHeader(SendOpen, 1, 0)

	// Size = 16 (0x10) in little-endian: 10 00 00 00
	assert.Equal(t, byte(0x10), data[0])
	assert.Equal(t, byte(0x00), data[1])
	assert.Equal(t, byte(0x00), data[2])
	assert.Equal(t, byte(0x00), data[3])

	// Version = 0x05 in little-endian: 05 00 00 00
	assert.Equal(t, byte(0x05), data[4])
	assert.Equal(t, byte(0x00), data[5])

	// Type = SendOpen | SendTypeMask = 0xf0000001: 01 00 00 f0
	assert.Equal(t, byte(0x01), data[8])
	assert.Equal(t, byte(0x00), data[9])
	assert.Equal(t, byte(0x00), data[10])
	assert.Equal(t, byte(0xf0), data[11])

	// ID = 1: 01 00 00 00
	assert.Equal(t, byte(0x01), data[12])
	assert.Equal(t, byte(0x00), data[13])
}
