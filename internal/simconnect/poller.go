package simconnect

import (
	"context"
	"log"
	"time"

	"github.com/eytandecker/flightsim-mcp/pkg/types"
)

// StateUpdater is implemented by state.Manager.
// Defined here (consuming side) to avoid import cycles.
type StateUpdater interface {
	Update(pos types.AircraftPosition)
	UpdateInstruments(inst types.FlightInstruments)
	UpdateEngine(eng types.EngineData)
	UpdateEnvironment(env types.Environment)
	UpdateAutopilot(ap types.AutopilotState)
}

// PollerConfig holds configuration for the Poller.
type PollerConfig struct {
	PollInterval time.Duration
}

// DefaultPollerConfig returns a PollerConfig with sensible defaults.
func DefaultPollerConfig() PollerConfig {
	return PollerConfig{PollInterval: 500 * time.Millisecond}
}

// Poller manages periodic SimConnect data requests and feeds updates to a StateUpdater.
type Poller struct {
	client  *Client
	updater StateUpdater
	cfg     PollerConfig
}

// NewPoller creates a Poller backed by the given client and updater.
func NewPoller(client *Client, updater StateUpdater, cfg PollerConfig) *Poller {
	return &Poller{client: client, updater: updater, cfg: cfg}
}

// dataGroup pairs a definition ID with a SimVar slice for registration.
type dataGroup struct {
	defID uint32
	vars  []SimVarDef
}

// allGroups lists all data groups to register and poll.
var allGroups = []dataGroup{
	{DefIDPosition, PositionSimVars},
	{DefIDInstruments, InstrumentsSimVars},
	{DefIDEngine, EngineSimVars},
	{DefIDEnvironment, EnvironmentSimVars},
	{DefIDAutopilot, AutopilotSimVars},
}

// RegisterSimVars calls AddToDataDefinition for each var in all data groups.
func (p *Poller) RegisterSimVars() error {
	for _, g := range allGroups {
		for _, sv := range g.vars {
			if err := p.client.AddToDataDefinition(g.defID, sv); err != nil {
				return err
			}
		}
	}
	return nil
}

// requestIDs maps definition IDs to request IDs for polling.
var requestIDs = []struct {
	defID uint32
	reqID uint32
}{
	{DefIDPosition, ReqIDPosition},
	{DefIDInstruments, ReqIDInstruments},
	{DefIDEngine, ReqIDEngine},
	{DefIDEnvironment, ReqIDEnvironment},
	{DefIDAutopilot, ReqIDAutopilot},
}

// Start blocks, sending periodic RequestData messages and processing responses.
// It exits when ctx is canceled or the connection is closed.
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
			for _, r := range requestIDs {
				if err := p.client.RequestData(r.defID, ObjectIDUser, r.reqID); err != nil {
					return err
				}
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
			p.dispatchPayload(h.ID, data)
		case MsgException:
			log.Printf("simconnect: received exception message (id=%d)", h.ID)
		}
	}
}

// dispatchPayload parses and routes a SimObjectData payload by request ID.
func (p *Poller) dispatchPayload(reqID uint32, data []byte) {
	switch reqID {
	case ReqIDPosition:
		pos, err := ParsePositionPayload(data)
		if err != nil {
			log.Printf("simconnect: parse position payload: %v", err)
			return
		}
		p.updater.Update(pos)
	case ReqIDInstruments:
		inst, err := ParseInstrumentsPayload(data)
		if err != nil {
			log.Printf("simconnect: parse instruments payload: %v", err)
			return
		}
		p.updater.UpdateInstruments(inst)
	case ReqIDEngine:
		eng, err := ParseEnginePayload(data)
		if err != nil {
			log.Printf("simconnect: parse engine payload: %v", err)
			return
		}
		p.updater.UpdateEngine(eng)
	case ReqIDEnvironment:
		env, err := ParseEnvironmentPayload(data)
		if err != nil {
			log.Printf("simconnect: parse environment payload: %v", err)
			return
		}
		p.updater.UpdateEnvironment(env)
	case ReqIDAutopilot:
		ap, err := ParseAutopilotPayload(data)
		if err != nil {
			log.Printf("simconnect: parse autopilot payload: %v", err)
			return
		}
		p.updater.UpdateAutopilot(ap)
	}
}
