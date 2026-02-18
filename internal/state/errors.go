package state

import "errors"

// ErrStale is returned when position data has not been updated within the stale threshold.
var ErrStale = errors.New("state: position data is stale")
