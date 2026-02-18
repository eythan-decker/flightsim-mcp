package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eytandecker/flightsim-mcp/internal/config"
	internalmcp "github.com/eytandecker/flightsim-mcp/internal/mcp"
	"github.com/eytandecker/flightsim-mcp/internal/simconnect"
	"github.com/eytandecker/flightsim-mcp/internal/state"
)

func main() {
	if err := run(); err != nil {
		log.Printf("MCP server exited: %v", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	mgr := state.NewManager(cfg.Polling.StaleThreshold)
	mcpServer := internalmcp.NewServer(mgr)

	go runPollerLoop(ctx, cfg, mgr)

	if err := mcpServer.Run(ctx); !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

// runPollerLoop connects to SimConnect and polls for data, retrying with
// exponential backoff (1s → 30s cap) on failure.
func runPollerLoop(ctx context.Context, cfg config.Config, mgr *state.Manager) {
	backoff := time.Second
	const maxBackoff = 30 * time.Second

	for {
		if err := ctx.Err(); err != nil {
			return
		}

		if err := runPoller(ctx, cfg, mgr); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			log.Printf("simconnect: disconnected: %v (retrying in %s)", err, backoff)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// runPoller creates a client, registers SimVars, and runs the polling loop.
// Returns when the connection is lost or ctx is done.
func runPoller(ctx context.Context, cfg config.Config, mgr *state.Manager) error {
	client := simconnect.NewClient(simconnect.Config{
		Host:    cfg.SimConnect.Host,
		Port:    cfg.SimConnect.Port,
		Timeout: cfg.SimConnect.Timeout,
		AppName: cfg.SimConnect.AppName,
	})

	if err := client.Connect(ctx); err != nil {
		return err
	}
	defer client.Close()

	pollerCfg := simconnect.PollerConfig{PollInterval: cfg.Polling.Interval}
	poller := simconnect.NewPoller(client, mgr, pollerCfg)

	if err := poller.RegisterSimVars(); err != nil {
		return err
	}

	return poller.Start(ctx)
}
