package simconnect

import (
	"encoding/binary"
	"fmt"
)

const (
	HeaderSize      = 16
	ProtocolVersion = 4

	MsgOpen               = 0x0001
	MsgClose              = 0x0002
	MsgRequestData        = 0x0003
	MsgSetDataDefinition  = 0x0004
	MsgAddToDataDef       = 0x0005
	MsgSimObjectData      = 0x0100
	MsgException          = 0x0101
)

// Header represents a SimConnect message header.
type Header struct {
	Size    uint32
	Version uint32
	Type    uint32
	ID      uint32
}

// EncodeHeader builds a 16-byte little-endian header for a SimConnect message.
// The Size field is set to HeaderSize + payloadSize.
func EncodeHeader(msgType, msgID uint32, payloadSize int) []byte {
	buf := make([]byte, HeaderSize)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(HeaderSize)+uint32(payloadSize))
	binary.LittleEndian.PutUint32(buf[4:8], ProtocolVersion)
	binary.LittleEndian.PutUint32(buf[8:12], msgType)
	binary.LittleEndian.PutUint32(buf[12:16], msgID)
	return buf
}

// DecodeHeader parses a 16-byte little-endian header from raw bytes.
func DecodeHeader(data []byte) (Header, error) {
	if len(data) < HeaderSize {
		return Header{}, fmt.Errorf("header too short: got %d bytes, need %d", len(data), HeaderSize)
	}
	return Header{
		Size:    binary.LittleEndian.Uint32(data[0:4]),
		Version: binary.LittleEndian.Uint32(data[4:8]),
		Type:    binary.LittleEndian.Uint32(data[8:12]),
		ID:      binary.LittleEndian.Uint32(data[12:16]),
	}, nil
}
