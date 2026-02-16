package types

import "fmt"

// SimulatorError wraps errors from the simulator with additional context.
type SimulatorError struct {
	Err         error
	Message     string
	Recoverable bool
}

func (e *SimulatorError) Error() string {
	return fmt.Sprintf("simulator error: %s: %v", e.Message, e.Err)
}

func (e *SimulatorError) Unwrap() error {
	return e.Err
}
