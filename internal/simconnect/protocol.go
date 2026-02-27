package simconnect

import (
	"encoding/binary"
	"fmt"
)

const (
	SendHeaderSize = 16 // 4 fields: Size, Version, Type, ID
	RecvHeaderSize = 12 // 3 fields: Size, Version, Type (no ID)

	ProtocolVersion uint32 = 0x05 // KittyHawk (MSFS 2024)

	// SendTypeMask is OR'd into the type field on all outgoing packets.
	SendTypeMask uint32 = 0xf0000000

	// Send types (raw values — mask applied in EncodeSendHeader).
	SendOpen         uint32 = 0x01
	SendClose        uint32 = 0x02
	SendAddToDataDef uint32 = 0x0c
	SendRequestData  uint32 = 0x0e

	// Receive types (no mask).
	RecvException     uint32 = 0x01
	RecvOpen          uint32 = 0x02
	RecvSimObjectData uint32 = 0x08

	// KittyHawk OPEN version constants.
	KHMajor      uint32 = 11
	KHMinor      uint32 = 0
	KHBuildMajor uint32 = 62651
	KHBuildMinor uint32 = 3
	KHAlias             = "HK"
)

// SendHeader represents an outgoing SimConnect message header (16 bytes).
type SendHeader struct {
	Size    uint32
	Version uint32
	Type    uint32 // raw type with mask already applied
	ID      uint32
}

// RecvHeader represents an incoming SimConnect message header (12 bytes, no ID).
type RecvHeader struct {
	Size    uint32
	Version uint32
	Type    uint32
}

// EncodeSendHeader builds a 16-byte little-endian header for an outgoing message.
// The mask is applied automatically to msgType.
func EncodeSendHeader(msgType, msgID uint32, payloadSize int) []byte {
	buf := make([]byte, SendHeaderSize)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(SendHeaderSize)+uint32(payloadSize)) // #nosec G115 -- payloadSize is a Go slice length, always non-negative
	binary.LittleEndian.PutUint32(buf[4:8], ProtocolVersion)
	binary.LittleEndian.PutUint32(buf[8:12], msgType|SendTypeMask)
	binary.LittleEndian.PutUint32(buf[12:16], msgID)
	return buf
}

// DecodeRecvHeader parses a 12-byte little-endian header from raw bytes.
func DecodeRecvHeader(data []byte) (RecvHeader, error) {
	if len(data) < RecvHeaderSize {
		return RecvHeader{}, fmt.Errorf("header too short: got %d bytes, need %d", len(data), RecvHeaderSize)
	}
	return RecvHeader{
		Size:    binary.LittleEndian.Uint32(data[0:4]),
		Version: binary.LittleEndian.Uint32(data[4:8]),
		Type:    binary.LittleEndian.Uint32(data[8:12]),
	}, nil
}
