package simconnect

import (
	"context"
	"log"
	"time"

	"github.com/eytandecker/flightsim-mcp/pkg/types"
)

// PositionUpdater is implemented by state.Manager.
// Defined here (consuming side) to avoid import cycles.
type PositionUpdater interface {
	Update(pos types.AircraftPosition)
}

// PollerConfig holds configuration for the Poller.
type PollerConfig struct {
	PollInterval time.Duration
}

// DefaultPollerConfig returns a PollerConfig with sensible defaults.
func DefaultPollerConfig() PollerConfig {
	return PollerConfig{PollInterval: 500 * time.Millisecond}
}

// Poller manages periodic SimConnect data requests and feeds updates to a PositionUpdater.
type Poller struct {
	client  *Client
	updater PositionUpdater
	cfg     PollerConfig
}

// NewPoller creates a Poller backed by the given client and updater.
func NewPoller(client *Client, updater PositionUpdater, cfg PollerConfig) *Poller {
	return &Poller{client: client, updater: updater, cfg: cfg}
}

// RegisterSimVars calls AddToDataDefinition for each var in PositionSimVars.
func (p *Poller) RegisterSimVars() error {
	for _, sv := range PositionSimVars {
		if err := p.client.AddToDataDefinition(DefIDPosition, sv); err != nil {
			return err
		}
	}
	return nil
}

// Start blocks, sending periodic RequestData messages and processing responses.
// It exits when ctx is cancelled or the connection is closed.
func (p *Poller) Start(ctx context.Context) error {
	interval := p.cfg.PollInterval
	if interval == 0 {
		interval = 500 * time.Millisecond
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	done := make(chan error, 1)
	go p.readLoop(done)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-done:
			return err
		case <-ticker.C:
			if err := p.client.RequestData(DefIDPosition, ObjectIDUser, ReqIDPosition); err != nil {
				return err
			}
		}
	}
}

// readLoop reads responses from SimConnect and dispatches them to the updater.
func (p *Poller) readLoop(done chan<- error) {
	for {
		h, data, err := p.client.ReadNext()
		if err != nil {
			done <- err
			return
		}
		switch h.Type {
		case MsgSimObjectData:
			if h.ID == ReqIDPosition {
				pos, err := ParsePositionPayload(data)
				if err != nil {
					log.Printf("simconnect: parse position payload: %v", err)
					continue
				}
				p.updater.Update(pos)
			}
		case MsgException:
			log.Printf("simconnect: received exception message (id=%d)", h.ID)
		}
	}
}
