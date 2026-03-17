# FlightSim-MCP

[![CI](https://github.com/eytandecker/flightsim-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/eytandecker/flightsim-mcp/actions/workflows/ci.yml)
![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)
![MSFS 2024](https://img.shields.io/badge/MSFS-2024-0078D4?logo=microsoftflightsimulator&logoColor=white)

A [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server that connects to Microsoft Flight Simulator 2024 via SimConnect, giving AI assistants real-time access to aircraft state, flight instruments, engine data, weather, and autopilot status.

```
┌──────────────┐    TCP/4500     ┌──────────────────┐    STDIO     ┌──────────────┐
│   MSFS 2024  │◄───────────────►│  flightsim-mcp   │◄────────────►│ Claude Code  │
│  (Windows)   │   SimConnect    │  (Go, any OS)    │     MCP      │ Claude Desktop│
└──────────────┘                 └──────────────────┘              └──────────────┘
```

Ask your AI assistant things like:
- *"What's my current altitude and heading?"*
- *"Check my engine N1 and EGT values"*
- *"What are the wind conditions right now?"*
- *"Is the autopilot engaged? What modes are active?"*

## Quick Start

### Prerequisites

- **Go 1.24+** — [install](https://go.dev/dl/)
- **MSFS 2024** running with SimConnect enabled (the server starts without it and auto-reconnects)
- **Claude Code** or **Claude Desktop** — [install Claude Code](https://docs.anthropic.com/en/docs/claude-code)

### Build

```bash
git clone https://github.com/eytandecker/flightsim-mcp.git
cd flightsim-mcp
make build
```

### Configure

Create or edit `.mcp.json` in the project root with your SimConnect host IP (the Windows machine running MSFS):

```json
{
  "mcpServers": {
    "flightsim-mcp": {
      "command": "make",
      "args": ["run"],
      "env": {
        "SIMCONNECT_HOST": "192.168.1.100",
        "SIMCONNECT_PORT": "4500"
      }
    }
  }
}
```

For **Claude Desktop**, add this to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "flightsim-mcp": {
      "command": "/absolute/path/to/flightsim-mcp/bin/flightsim-mcp",
      "env": {
        "SIMCONNECT_HOST": "192.168.1.100",
        "SIMCONNECT_PORT": "4500"
      }
    }
  }
}
```

### Verify

```bash
# Claude Code — should show flightsim-mcp in the list
claude mcp list
```

## Available Tools

| Tool | Description |
|------|-------------|
| `get_aircraft_position` | Latitude, longitude, altitude (MSL/AGL), heading, airspeed, ground speed, vertical speed. Optional pitch/bank via `include_attitude`. |
| `get_flight_instruments` | Indicated altitude, altimeter setting, vertical speed, airspeed (IAS/TAS/Mach), heading indicator, turn coordinator, attitude. |
| `get_engine_data` | Throttle position, RPM, N1/N2, fuel flow, EGT, oil temp/pressure for up to 2 engines. Total and per-tank fuel quantities. |
| `get_environment` | Wind speed and direction, temperature, barometric pressure, visibility, precipitation state, local and Zulu time. |
| `get_autopilot_state` | AP master, heading/altitude/VS/airspeed hold modes, NAV1 and approach modes, flight director, and all target values. |

All tools return structured JSON. When the simulator is not connected or data is stale, tools return an error response with a diagnostic code (`SIMULATOR_NOT_CONNECTED`, `DATA_STALE`) and a recovery suggestion — the LLM uses these to inform the user gracefully.

## SimConnect Setup (MSFS 2024)

FlightSim-MCP connects to MSFS 2024 over TCP using the SimConnect binary wire protocol. No SimConnect SDK installation is needed on the machine running the MCP server.

On the **Windows machine running MSFS 2024**:

1. Locate or create `SimConnect.xml` in your MSFS config directory (typically `%APPDATA%\Microsoft Flight Simulator 2024\`).

2. Ensure it includes a TCP configuration block:

```xml
<SimConnect.Comm>
  <Protocol>IPv4</Protocol>
  <Scope>global</Scope>
  <Port>4500</Port>
  <MaxClients>64</MaxClients>
</SimConnect.Comm>
```

3. Allow inbound TCP connections on port 4500 in Windows Firewall from the machine running the MCP server.

4. Start MSFS 2024 — SimConnect listens automatically.

The MCP server connects over the network with exponential backoff (1s → 30s cap), so it can be started before or after the simulator.

## Configuration

All settings are environment variables. Set them in `.mcp.json` or export them in your shell.

| Variable | Default | Description |
|----------|---------|-------------|
| `SIMCONNECT_HOST` | `192.168.10.100` | IP of the Windows machine running MSFS |
| `SIMCONNECT_PORT` | `4500` | SimConnect TCP port |
| `SIMCONNECT_TIMEOUT` | `10s` | TCP connection timeout |
| `SIMCONNECT_APP_NAME` | `flightsim-mcp` | App name in the SimConnect handshake |
| `POLL_INTERVAL` | `500ms` | How often to request fresh data from SimConnect |
| `STALE_THRESHOLD` | `5s` | Data older than this triggers a stale-data error |

## Project Structure

```
flightsim-mcp/
├── cmd/flightsim-mcp/       # Entry point, signal handling, reconnect loop
├── internal/
│   ├── config/              # Environment variable loader
│   ├── mcp/                 # MCP server, tool definitions, handlers
│   ├── simconnect/          # SimConnect TCP client, wire protocol, SimVar defs, poller
│   └── state/               # Thread-safe state cache with staleness detection
├── pkg/types/               # Shared data types (position, instruments, engine, etc.)
├── deploy/                  # Docker and Kubernetes manifests (planned)
├── docs/                    # Additional documentation
├── .github/workflows/       # CI pipeline (lint, test, security, build)
└── Makefile                 # Build, test, lint, coverage commands
```

## Development

### Build & Test

```bash
make build        # Build binary to bin/flightsim-mcp
make test         # Run tests with race detection
make lint         # Run golangci-lint
make coverage     # Generate coverage report (coverage.html)
make all          # fmt + vet + lint + test + build
```

### Running Locally

```bash
# Set your SimConnect host and run
SIMCONNECT_HOST=192.168.1.100 make run
```

The server communicates via STDIO (stdin/stdout), which is how MCP clients like Claude Code interact with it. Log output goes to stderr.


## How It Works

1. **Connect** — The server dials the SimConnect TCP endpoint on your Windows machine and performs the KittyHawk (MSFS 2024) binary handshake.

2. **Register** — 63 simulation variables across 5 groups (position, instruments, engine, environment, autopilot) are registered with SimConnect via `AddToDataDefinition`.

3. **Poll** — A background poller requests fresh data at the configured interval. A read loop receives SimConnect responses and dispatches them to the correct parser by request ID.

4. **Cache** — Parsed data is stored in a thread-safe state manager with per-group staleness tracking.

5. **Serve** — When an MCP client calls a tool, the handler reads the cached state and returns structured JSON. If data is stale or the simulator isn't connected, the response includes a diagnostic error code.

## Troubleshooting

| Problem | Solution |
|---------|----------|
| Tools return `SIMULATOR_NOT_CONNECTED` | Start MSFS 2024 — the server auto-reconnects |
| Tools return `DATA_STALE` | Check network connectivity; increase `STALE_THRESHOLD` if on a slow link |
| `flightsim-mcp` not in `claude mcp list` | Run from the project root (where `.mcp.json` lives); run `make build` |
| Connection refused on port 4500 | Verify `SimConnect.xml` config and Windows Firewall rules |

See [docs/claude-code-setup.md](docs/claude-code-setup.md) for a detailed setup and troubleshooting guide.
