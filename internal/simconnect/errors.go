package simconnect

import "errors"

var (
	ErrNotConnected     = errors.New("simconnect: not connected")
	ErrTimeout          = errors.New("simconnect: connection timeout")
	ErrInvalidSimVar    = errors.New("simconnect: invalid simvar")
	ErrConnectionRefused = errors.New("simconnect: connection refused")
)
