# FlightSim-MCP Server Specification

> A Model Context Protocol (MCP) server that exposes Microsoft Flight Simulator 2024 aircraft state data to LLM-powered applications.

**Version:** 1.0.0-MVP
**Target Aircraft:** Airbus A320 (CFM Engine)
**Operations:** READ-ONLY
**Primary LLM:** Claude (configurable)

---

## Table of Contents

1. [Overview](#1-overview)
2. [Architecture](#2-architecture)
3. [Project Structure](#3-project-structure)
4. [SimConnect Integration](#4-simconnect-integration)
5. [MCP Server Implementation](#5-mcp-server-implementation)
6. [Coding Standards](#6-coding-standards)
7. [Testing Strategy](#7-testing-strategy)
8. [Security Requirements](#8-security-requirements)
9. [CI/CD Pipeline](#9-cicd-pipeline)
10. [Deployment](#10-deployment)
11. [Configuration](#11-configuration)
12. [Logging & Monitoring](#12-logging--monitoring)
13. [Implementation Phases](#13-implementation-phases)
14. [CLAUDE.md Template](#14-claudemd-template)
15. [Starter Prompt](#15-starter-prompt)

---

## 1. Overview

### 1.1 What This Project Is

FlightSim-MCP is a Go-based MCP server that connects to Microsoft Flight Simulator 2024 via the SimConnect SDK and exposes aircraft state data through the Model Context Protocol. This enables LLM applications (primarily Claude) to query real-time flight data for use cases like:

- Virtual co-pilot assistance
- Flight training feedback
- Procedure verification
- Real-time flight monitoring dashboards

### 1.2 MVP Scope

| Aspect | MVP Scope |
|--------|-----------|
| Aircraft | A320 with CFM engine only |
| Operations | **READ-ONLY** (no write operations) |
| SimVars | ~70 variables across 9 categories |
| Transport | STDIO (Claude Desktop), HTTP (remote) |
| Deployment | Docker container connecting to Windows host |

### 1.3 Target Users

- Flight simulation enthusiasts wanting AI-assisted flying
- Developers building LLM-powered aviation applications
- Homelab operators running K3s/Fleet GitOps infrastructure

### 1.4 Key Constraints

1. **No mature Go SimConnect library exists** - Must implement binary wire protocol
2. **SimConnect runs on Windows only** - Server runs in container, connects via TCP
3. **READ-ONLY for MVP** - Write operations deferred due to safety concerns
4. **Network latency** - Container-to-host communication adds ~1-5ms

---

## 2. Architecture

### 2.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           WINDOWS HOST                                   │
│  ┌─────────────────────┐     ┌──────────────────────────────────────┐  │
│  │  MS Flight Sim 2024 │────▶│  SimConnect Server (Port 4500)       │  │
│  │  (A320 Aircraft)    │     │  Configured via SimConnect.xml       │  │
│  └─────────────────────┘     └──────────────────┬───────────────────┘  │
│                                                  │ TCP/IPv4             │
└──────────────────────────────────────────────────┼──────────────────────┘
                                                   │
                    ┌──────────────────────────────┼──────────────────────┐
                    │          KUBERNETES (K3s)    │                      │
                    │  ┌───────────────────────────▼───────────────────┐  │
                    │  │         FlightSim-MCP Container               │  │
                    │  │  ┌─────────────────────────────────────────┐  │  │
                    │  │  │  SimConnect Client (Wire Protocol)      │  │  │
                    │  │  │  - Connection management                │  │  │
                    │  │  │  - Binary message encoding/decoding     │  │  │
                    │  │  │  - SimVar subscription & polling        │  │  │
                    │  │  └────────────────────┬────────────────────┘  │  │
                    │  │                       │                       │  │
                    │  │  ┌────────────────────▼────────────────────┐  │  │
                    │  │  │  State Manager                          │  │  │
                    │  │  │  - Concurrent-safe state cache          │  │  │
                    │  │  │  - Polling orchestration                │  │  │
                    │  │  │  - Data transformation                  │  │  │
                    │  │  └────────────────────┬────────────────────┘  │  │
                    │  │                       │                       │  │
                    │  │  ┌────────────────────▼────────────────────┐  │  │
                    │  │  │  MCP Server (go-sdk)                    │  │  │
                    │  │  │  - Tool handlers                        │  │  │
                    │  │  │  - STDIO / HTTP transport               │  │  │
                    │  │  │  - Request validation                   │  │  │
                    │  │  └────────────────────┬────────────────────┘  │  │
                    │  └───────────────────────┼───────────────────────┘  │
                    │                          │                          │
                    │  ┌───────────────────────▼───────────────────────┐  │
                    │  │  Traefik Ingress (192.168.10.35)             │  │
                    │  │  moria-lab.com with cert-manager SSL         │  │
                    │  └───────────────────────┬───────────────────────┘  │
                    └──────────────────────────┼──────────────────────────┘
                                               │
                    ┌──────────────────────────▼──────────────────────────┐
                    │              LLM CLIENT (Claude)                     │
                    │  - Claude Desktop (STDIO)                           │
                    │  - API Integration (HTTP/SSE)                       │
                    └─────────────────────────────────────────────────────┘
```

### 2.2 Component Responsibilities

| Component | Responsibility |
|-----------|----------------|
| **SimConnect Client** | TCP connection, wire protocol, message framing |
| **State Manager** | Caching, polling tiers, thread-safe access |
| **MCP Server** | Tool registration, request handling, response formatting |
| **Transport Layer** | STDIO for desktop, HTTP/SSE for remote |

### 2.3 Data Flow

```
1. LLM calls MCP tool (e.g., get_aircraft_position)
2. MCP Server validates request
3. State Manager returns cached data (or triggers poll)
4. SimConnect Client sends binary request to MSFS
5. MSFS returns SimVar data via TCP
6. Data transformed and returned to LLM as JSON
```

---

## 3. Project Structure

```
flightsim-mcp/
├── .github/
│   └── workflows/
│       ├── ci.yml                  # CI workflow (push/PR)
│       └── release.yml             # Release workflow (tags)
├── cmd/
│   └── flightsim-mcp/
│       └── main.go                 # Entry point, CLI flags, DI setup
├── internal/
│   ├── simconnect/
│   │   ├── client.go               # TCP connection, wire protocol
│   │   ├── client_test.go
│   │   ├── protocol.go             # Message types, header encoding
│   │   ├── protocol_test.go
│   │   ├── simvars.go              # SimVar definitions & registry
│   │   ├── simvars_test.go
│   │   └── errors.go               # SimConnect-specific errors
│   ├── state/
│   │   ├── manager.go              # Concurrent state cache
│   │   ├── manager_test.go
│   │   ├── poller.go               # Polling orchestration
│   │   └── poller_test.go
│   ├── mcp/
│   │   ├── server.go               # MCP server setup
│   │   ├── server_test.go
│   │   ├── tools.go                # Tool definitions
│   │   ├── tools_test.go
│   │   ├── handlers.go             # Tool handler implementations
│   │   └── handlers_test.go
│   ├── transport/
│   │   ├── stdio.go                # STDIO transport for Claude Desktop
│   │   ├── http.go                 # HTTP/SSE transport
│   │   └── transport_test.go
│   └── config/
│       ├── config.go               # Configuration struct & loading
│       └── config_test.go
├── pkg/
│   └── types/
│       ├── aircraft.go             # Shared types (Position, Engine, etc.)
│       └── errors.go               # Public error types
├── deploy/
│   ├── docker/
│   │   └── Dockerfile
│   └── fleet/
│       ├── fleet.yaml
│       ├── Chart.yaml
│       ├── values.yaml
│       └── templates/
│           ├── deployment.yaml
│           ├── service.yaml
│           ├── configmap.yaml
│           └── networkpolicy.yaml
├── scripts/
│   ├── simconnect-config/
│   │   └── SimConnect.xml          # Template for Windows host
│   └── test-connection.sh
├── docs/
│   └── simvars.md                  # Complete SimVar reference
├── .golangci.yml                   # Linter configuration
├── go.mod
├── go.sum
├── Makefile
├── CLAUDE.md
└── README.md
```

### 3.1 Directory Explanations

| Directory | Purpose |
|-----------|---------|
| `.github/` | GitHub Actions CI/CD workflows |
| `cmd/` | Application entry points |
| `internal/` | Private packages, not importable externally |
| `pkg/` | Public packages, safe for external import |
| `deploy/` | Deployment manifests (Docker, Fleet/K8s) |
| `scripts/` | Helper scripts and configuration templates |
| `docs/` | Extended documentation |

---

## 4. SimConnect Integration

### 4.1 Connection Approach

**TCP/IPv4 Network Connection** (not named pipes - those are local only)

```go
// Connection configuration
type SimConnectConfig struct {
    Host    string        // Windows host IP (e.g., "192.168.10.100")
    Port    int           // Default 4500
    Timeout time.Duration // Connection timeout
    AppName string        // "FlightSim-MCP"
}
```

### 4.2 Wire Protocol

SimConnect uses a binary protocol with the following structure:

```
┌─────────────────────────────────────────────────────────┐
│                    MESSAGE HEADER (16 bytes)             │
├──────────────┬──────────────┬──────────────┬────────────┤
│  Size (4B)   │  Version (4B)│  Type (4B)   │  ID (4B)   │
├──────────────┴──────────────┴──────────────┴────────────┤
│                    MESSAGE PAYLOAD                       │
│              (Variable length, Type-dependent)           │
└─────────────────────────────────────────────────────────┘
```

#### Header Constants

```go
const (
    HeaderSize       = 16
    ProtocolVersion  = 4  // MSFS 2024

    // Message Types (subset for MVP)
    MSG_OPEN                    = 0x0001
    MSG_CLOSE                   = 0x0002
    MSG_REQUEST_DATA            = 0x0003
    MSG_SET_DATA_DEFINITION     = 0x0004
    MSG_ADD_TO_DATA_DEFINITION  = 0x0005
    MSG_SIMOBJECT_DATA          = 0x0100
    MSG_EXCEPTION               = 0x0101
)
```

#### Message Encoding Example

```go
func (c *Client) encodeHeader(msgType, msgID uint32, payloadSize int) []byte {
    buf := make([]byte, HeaderSize)
    binary.LittleEndian.PutUint32(buf[0:4], uint32(HeaderSize+payloadSize))
    binary.LittleEndian.PutUint32(buf[4:8], ProtocolVersion)
    binary.LittleEndian.PutUint32(buf[8:12], msgType)
    binary.LittleEndian.PutUint32(buf[12:16], msgID)
    return buf
}
```

### 4.3 SimVar Categories & Definitions

#### Category 1: Position & Movement (Fast Poll - 50ms)

| SimVar | Unit | Type | Description |
|--------|------|------|-------------|
| PLANE LATITUDE | degrees | FLOAT64 | Aircraft latitude |
| PLANE LONGITUDE | degrees | FLOAT64 | Aircraft longitude |
| PLANE ALTITUDE | feet | FLOAT64 | MSL altitude |
| PLANE ALT ABOVE GROUND | feet | FLOAT64 | AGL altitude |
| PLANE HEADING DEGREES TRUE | degrees | FLOAT64 | True heading |
| PLANE HEADING DEGREES MAGNETIC | degrees | FLOAT64 | Magnetic heading |
| PLANE PITCH DEGREES | degrees | FLOAT64 | Pitch attitude |
| PLANE BANK DEGREES | degrees | FLOAT64 | Bank angle |
| AIRSPEED INDICATED | knots | FLOAT64 | IAS |
| AIRSPEED TRUE | knots | FLOAT64 | TAS |
| GROUND VELOCITY | knots | FLOAT64 | Ground speed |
| VERTICAL SPEED | feet/minute | FLOAT64 | Vertical speed |

#### Category 2: Autopilot State (Medium Poll - 200ms)

| SimVar | Unit | Type | Description |
|--------|------|------|-------------|
| AUTOPILOT MASTER | bool | INT32 | AP master engaged |
| AUTOPILOT HEADING LOCK | bool | INT32 | Heading mode |
| AUTOPILOT HEADING LOCK DIR | degrees | FLOAT64 | Selected heading |
| AUTOPILOT ALTITUDE LOCK | bool | INT32 | Altitude hold |
| AUTOPILOT ALTITUDE LOCK VAR | feet | FLOAT64 | Selected altitude |
| AUTOPILOT VERTICAL HOLD | bool | INT32 | VS mode |
| AUTOPILOT VERTICAL HOLD VAR | feet/minute | FLOAT64 | Selected VS |
| AUTOPILOT AIRSPEED HOLD | bool | INT32 | Speed mode |
| AUTOPILOT AIRSPEED HOLD VAR | knots | FLOAT64 | Selected speed |
| AUTOPILOT APPROACH HOLD | bool | INT32 | APPR mode |
| AUTOPILOT NAV1 LOCK | bool | INT32 | NAV mode |
| AUTOPILOT FLIGHT DIRECTOR ACTIVE | bool | INT32 | FD active |

#### Category 3: Engine Status - CFM (Medium Poll - 200ms)

| SimVar | Unit | Type | Description |
|--------|------|------|-------------|
| ENG N1 RPM:1 | percent | FLOAT64 | Engine 1 N1 |
| ENG N1 RPM:2 | percent | FLOAT64 | Engine 2 N1 |
| ENG N2 RPM:1 | percent | FLOAT64 | Engine 1 N2 |
| ENG N2 RPM:2 | percent | FLOAT64 | Engine 2 N2 |
| ENG FUEL FLOW GPH:1 | gallons/hour | FLOAT64 | Engine 1 FF |
| ENG FUEL FLOW GPH:2 | gallons/hour | FLOAT64 | Engine 2 FF |
| ENG OIL PRESSURE:1 | psi | FLOAT64 | Engine 1 oil press |
| ENG OIL PRESSURE:2 | psi | FLOAT64 | Engine 2 oil press |
| ENG OIL TEMPERATURE:1 | celsius | FLOAT64 | Engine 1 oil temp |
| ENG OIL TEMPERATURE:2 | celsius | FLOAT64 | Engine 2 oil temp |
| ENG EXHAUST GAS TEMPERATURE:1 | celsius | FLOAT64 | Engine 1 EGT |
| ENG EXHAUST GAS TEMPERATURE:2 | celsius | FLOAT64 | Engine 2 EGT |
| TURB ENG ITT:1 | celsius | FLOAT64 | Engine 1 ITT |
| TURB ENG ITT:2 | celsius | FLOAT64 | Engine 2 ITT |

#### Category 4: Flight Controls (Medium Poll - 200ms)

| SimVar | Unit | Type | Description |
|--------|------|------|-------------|
| ELEVATOR POSITION | position | FLOAT64 | Elevator deflection |
| AILERON POSITION | position | FLOAT64 | Aileron deflection |
| RUDDER POSITION | position | FLOAT64 | Rudder deflection |
| FLAPS HANDLE INDEX | number | INT32 | Flap setting |
| FLAPS HANDLE PERCENT | percent | FLOAT64 | Flap position |
| SPOILERS HANDLE POSITION | percent | FLOAT64 | Spoiler position |
| SPOILERS ARMED | bool | INT32 | Spoilers armed |
| GEAR HANDLE POSITION | bool | INT32 | Gear handle |
| GEAR POSITION:0 | percent | FLOAT64 | Nose gear pos |
| GEAR POSITION:1 | percent | FLOAT64 | Left gear pos |
| GEAR POSITION:2 | percent | FLOAT64 | Right gear pos |

#### Category 5: Fuel System (Slow Poll - 1000ms)

| SimVar | Unit | Type | Description |
|--------|------|------|-------------|
| FUEL TOTAL QUANTITY | gallons | FLOAT64 | Total fuel |
| FUEL TOTAL QUANTITY WEIGHT | pounds | FLOAT64 | Total fuel weight |
| FUEL LEFT QUANTITY | gallons | FLOAT64 | Left tank |
| FUEL RIGHT QUANTITY | gallons | FLOAT64 | Right tank |
| FUEL CENTER QUANTITY | gallons | FLOAT64 | Center tank |

#### Category 6: Electrical (Slow Poll - 1000ms)

| SimVar | Unit | Type | Description |
|--------|------|------|-------------|
| ELECTRICAL MASTER BATTERY | bool | INT32 | Battery master |
| ELECTRICAL MAIN BUS VOLTAGE | volts | FLOAT64 | Main bus V |
| ELECTRICAL BATTERY LOAD | percent | FLOAT64 | Battery load |
| ELECTRICAL GENALT BUS VOLTAGE:1 | volts | FLOAT64 | Gen 1 bus V |
| ELECTRICAL GENALT BUS VOLTAGE:2 | volts | FLOAT64 | Gen 2 bus V |

#### Category 7: Hydraulics (Slow Poll - 1000ms)

| SimVar | Unit | Type | Description |
|--------|------|------|-------------|
| HYDRAULIC PRESSURE:1 | psi | FLOAT64 | Green system |
| HYDRAULIC PRESSURE:2 | psi | FLOAT64 | Blue system |
| HYDRAULIC PRESSURE:3 | psi | FLOAT64 | Yellow system |

#### Category 8: Pressurization (Slow Poll - 1000ms)

| SimVar | Unit | Type | Description |
|--------|------|------|-------------|
| PRESSURIZATION CABIN ALTITUDE | feet | FLOAT64 | Cabin altitude |
| PRESSURIZATION DIFF PRESSURE PSI | psi | FLOAT64 | Diff pressure |

#### Category 9: Navigation (Medium Poll - 200ms)

| SimVar | Unit | Type | Description |
|--------|------|------|-------------|
| NAV OBS:1 | degrees | FLOAT64 | VOR 1 OBS |
| NAV OBS:2 | degrees | FLOAT64 | VOR 2 OBS |
| NAV RADIAL:1 | degrees | FLOAT64 | VOR 1 radial |
| NAV RADIAL:2 | degrees | FLOAT64 | VOR 2 radial |
| NAV DME:1 | nautical miles | FLOAT64 | DME 1 distance |
| NAV DME:2 | nautical miles | FLOAT64 | DME 2 distance |
| ADF RADIAL:1 | degrees | FLOAT64 | ADF bearing |
| GPS GROUND SPEED | knots | FLOAT64 | GPS ground speed |
| GPS COURSE TO STEER | degrees | FLOAT64 | GPS DTK |

### 4.4 Polling Strategy

```go
type PollTier struct {
    Name     string
    Interval time.Duration
    SimVars  []string
}

var PollTiers = []PollTier{
    {
        Name:     "fast",
        Interval: 50 * time.Millisecond,
        SimVars:  []string{"PLANE LATITUDE", "PLANE LONGITUDE", ...},
    },
    {
        Name:     "medium",
        Interval: 200 * time.Millisecond,
        SimVars:  []string{"AUTOPILOT MASTER", "ENG N1 RPM:1", ...},
    },
    {
        Name:     "slow",
        Interval: 1000 * time.Millisecond,
        SimVars:  []string{"FUEL TOTAL QUANTITY", "HYDRAULIC PRESSURE:1", ...},
    },
}
```

### 4.5 SimConnect.xml (Windows Host Configuration)

Place at: `%APPDATA%\Microsoft Flight Simulator\SimConnect.xml`

```xml
<?xml version="1.0" encoding="Windows-1252"?>
<SimBase.Document Type="SimConnect" version="1,0">
    <Filename>SimConnect.xml</Filename>
    <SimConnect.Comm>
        <Protocol>IPv4</Protocol>
        <Scope>global</Scope>
        <Address>0.0.0.0</Address>
        <Port>4500</Port>
        <MaxClients>64</MaxClients>
        <MaxRecvSize>41088</MaxRecvSize>
        <DisableNagle>False</DisableNagle>
    </SimConnect.Comm>
</SimBase.Document>
```

### 4.6 Connection Management & Reconnection

SimConnect on the Windows host may not always be available. The MCP server must handle:
- **Startup without SimConnect**: MSFS not launched yet
- **Mid-session disconnection**: MSFS crash, network issue
- **Reconnection**: MSFS becomes available again

#### Architectural Decisions

| Scenario | Behavior |
|----------|----------|
| SimConnect unavailable at startup | Server starts anyway, tools return graceful errors |
| Connection lost mid-session | Automatic reconnection with exponential backoff |
| SimConnect becomes available | Connection established, data polling resumes |
| Data staleness | Track last update time, reject stale data |

#### Connection States

```go
type ConnectionState int32

const (
    StateDisconnected ConnectionState = iota  // Not connected
    StateConnecting                           // Connection attempt in progress
    StateConnected                            // Connected and healthy
    StateReconnecting                         // Was connected, attempting reconnect
)
```

#### Exponential Backoff Configuration

```go
type BackoffConfig struct {
    InitialInterval time.Duration  // 1 second
    MaxInterval     time.Duration  // 60 seconds
    Multiplier      float64        // 2.0
    MaxJitter       time.Duration  // 1 second (prevents thundering herd)
}
```

#### Connection Manager Pattern

```go
// RunConnectionManager handles connection lifecycle with automatic reconnection
func (c *Client) RunConnectionManager(ctx context.Context) error {
    currentInterval := c.backoff.InitialInterval

    for {
        select {
        case <-ctx.Done():
            c.Close()
            return ctx.Err()
        default:
        }

        // Attempt connection
        err := c.Connect(ctx)
        if err != nil {
            log.Warn().
                Err(err).
                Dur("retry_in", currentInterval).
                Msg("SimConnect connection failed, will retry")

            // Wait with backoff + jitter before retry
            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(c.addJitter(currentInterval)):
            }

            // Increase backoff interval
            currentInterval = min(
                time.Duration(float64(currentInterval) * c.backoff.Multiplier),
                c.backoff.MaxInterval,
            )
            continue
        }

        // Reset backoff on successful connection
        currentInterval = c.backoff.InitialInterval

        // Monitor connection health
        c.monitorConnection(ctx)

        // If we reach here, connection was lost
        log.Warn().Msg("SimConnect connection lost, attempting reconnection")
    }
}
```

#### Health Monitoring

```go
// monitorConnection periodically verifies connection is alive
func (c *Client) monitorConnection(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := c.ping(); err != nil {
                log.Warn().Err(err).Msg("SimConnect health check failed")
                c.Close()
                return
            }
        }
    }
}
```

---

## 5. MCP Server Implementation

### 5.1 Dependencies

```go
// go.mod
module github.com/yourusername/flightsim-mcp

go 1.22

require (
    github.com/modelcontextprotocol/go-sdk v0.1.0
    github.com/rs/zerolog v1.32.0
    go.opentelemetry.io/otel v1.24.0
    go.opentelemetry.io/otel/trace v1.24.0
)
```

### 5.2 MCP Tools (MVP)

#### Tool 1: get_aircraft_position

```go
var GetAircraftPositionTool = mcp.Tool{
    Name:        "get_aircraft_position",
    Description: "Get current aircraft position, altitude, heading, and speeds",
    InputSchema: mcp.Schema{
        Type: "object",
        Properties: map[string]mcp.Property{
            "include_attitude": {
                Type:        "boolean",
                Description: "Include pitch and bank angles",
                Default:     true,
            },
        },
    },
}

// Response structure
type AircraftPosition struct {
    Latitude       float64 `json:"latitude"`
    Longitude      float64 `json:"longitude"`
    AltitudeMSL    float64 `json:"altitude_msl_ft"`
    AltitudeAGL    float64 `json:"altitude_agl_ft"`
    HeadingTrue    float64 `json:"heading_true_deg"`
    HeadingMag     float64 `json:"heading_mag_deg"`
    IndicatedSpeed float64 `json:"indicated_airspeed_kt"`
    TrueSpeed      float64 `json:"true_airspeed_kt"`
    GroundSpeed    float64 `json:"ground_speed_kt"`
    VerticalSpeed  float64 `json:"vertical_speed_fpm"`
    // Optional attitude
    Pitch          *float64 `json:"pitch_deg,omitempty"`
    Bank           *float64 `json:"bank_deg,omitempty"`
    Timestamp      string   `json:"timestamp"`
}
```

#### Tool 2: get_autopilot_state

```go
var GetAutopilotStateTool = mcp.Tool{
    Name:        "get_autopilot_state",
    Description: "Get autopilot settings and active modes",
    InputSchema: mcp.Schema{
        Type:       "object",
        Properties: map[string]mcp.Property{},
    },
}

// Response structure
type AutopilotState struct {
    MasterEngaged     bool    `json:"master_engaged"`
    FlightDirector    bool    `json:"flight_director_active"`
    HeadingMode       bool    `json:"heading_mode"`
    SelectedHeading   float64 `json:"selected_heading_deg"`
    AltitudeHold      bool    `json:"altitude_hold"`
    SelectedAltitude  float64 `json:"selected_altitude_ft"`
    VerticalSpeedMode bool    `json:"vertical_speed_mode"`
    SelectedVS        float64 `json:"selected_vs_fpm"`
    SpeedMode         bool    `json:"speed_mode"`
    SelectedSpeed     float64 `json:"selected_speed_kt"`
    ApproachMode      bool    `json:"approach_mode"`
    NavMode           bool    `json:"nav_mode"`
    Timestamp         string  `json:"timestamp"`
}
```

#### Tool 3: get_engine_status

```go
var GetEngineStatusTool = mcp.Tool{
    Name:        "get_engine_status",
    Description: "Get engine parameters for both CFM engines",
    InputSchema: mcp.Schema{
        Type: "object",
        Properties: map[string]mcp.Property{
            "engine": {
                Type:        "integer",
                Description: "Engine number (1 or 2). Omit for both engines.",
                Enum:        []interface{}{1, 2},
            },
        },
    },
}

// Response structure
type EngineStatus struct {
    Engines   []EngineData `json:"engines"`
    Timestamp string       `json:"timestamp"`
}

type EngineData struct {
    Number      int     `json:"engine_number"`
    N1Percent   float64 `json:"n1_percent"`
    N2Percent   float64 `json:"n2_percent"`
    FuelFlowGPH float64 `json:"fuel_flow_gph"`
    OilPressure float64 `json:"oil_pressure_psi"`
    OilTemp     float64 `json:"oil_temp_celsius"`
    EGT         float64 `json:"egt_celsius"`
    ITT         float64 `json:"itt_celsius"`
}
```

#### Tool 4: get_systems_status

```go
var GetSystemsStatusTool = mcp.Tool{
    Name:        "get_systems_status",
    Description: "Get aircraft systems status (fuel, electrical, hydraulics, pressurization, flight controls)",
    InputSchema: mcp.Schema{
        Type: "object",
        Properties: map[string]mcp.Property{
            "systems": {
                Type:        "array",
                Description: "Systems to query. Options: fuel, electrical, hydraulics, pressurization, controls. Omit for all.",
                Items:       &mcp.Schema{Type: "string"},
            },
        },
    },
}

// Response structure
type SystemsStatus struct {
    Fuel          *FuelSystem          `json:"fuel,omitempty"`
    Electrical    *ElectricalSystem    `json:"electrical,omitempty"`
    Hydraulics    *HydraulicsSystem    `json:"hydraulics,omitempty"`
    Pressurization *PressurizationSystem `json:"pressurization,omitempty"`
    FlightControls *FlightControlsSystem `json:"flight_controls,omitempty"`
    Timestamp     string               `json:"timestamp"`
}
```

#### Tool 5: get_simulator_status

**Use this tool first** to check if the simulator is connected before querying flight data.

```go
var GetSimulatorStatusTool = mcp.Tool{
    Name:        "get_simulator_status",
    Description: "Check if Microsoft Flight Simulator is connected and data is available. Call this before other tools to verify simulator availability.",
    InputSchema: mcp.Schema{
        Type:       "object",
        Properties: map[string]mcp.Property{},
    },
}

// Response structure
type SimulatorStatusResponse struct {
    Connected        bool   `json:"connected"`
    ConnectionState  string `json:"connection_state"`
    DataAvailable    bool   `json:"data_available"`
    ConnectAttempts  int64  `json:"connect_attempts"`
    LastConnectedAt  string `json:"last_connected_at,omitempty"`
    LastError        string `json:"last_error,omitempty"`
    Message          string `json:"message"`
    Timestamp        string `json:"timestamp"`
}
```

**Example responses:**

```json
// Simulator not running
{
    "connected": false,
    "connection_state": "disconnected",
    "data_available": false,
    "connect_attempts": 5,
    "last_error": "connection refused",
    "message": "Flight simulator is not connected. Start Microsoft Flight Simulator 2024 to enable flight data access.",
    "timestamp": "2024-01-15T10:30:00Z"
}

// Simulator connected and ready
{
    "connected": true,
    "connection_state": "connected",
    "data_available": true,
    "connect_attempts": 1,
    "last_connected_at": "2024-01-15T10:25:00Z",
    "message": "Flight simulator is connected and data is available.",
    "timestamp": "2024-01-15T10:30:00Z"
}
```

### 5.3 Server Setup

```go
package mcp

import (
    "context"

    mcpsdk "github.com/modelcontextprotocol/go-sdk"
    "github.com/yourusername/flightsim-mcp/internal/state"
)

type Server struct {
    mcp          *mcpsdk.Server
    stateManager *state.Manager
    logger       zerolog.Logger
}

func NewServer(sm *state.Manager, logger zerolog.Logger) (*Server, error) {
    s := &Server{
        stateManager: sm,
        logger:       logger,
    }

    // Create MCP server
    server, err := mcpsdk.NewServer(mcpsdk.ServerConfig{
        Name:    "flightsim-mcp",
        Version: "1.0.0",
    })
    if err != nil {
        return nil, fmt.Errorf("creating MCP server: %w", err)
    }

    // Register tools
    server.RegisterTool(GetAircraftPositionTool, s.handleGetAircraftPosition)
    server.RegisterTool(GetAutopilotStateTool, s.handleGetAutopilotState)
    server.RegisterTool(GetEngineStatusTool, s.handleGetEngineStatus)
    server.RegisterTool(GetSystemsStatusTool, s.handleGetSystemsStatus)

    s.mcp = server
    return s, nil
}

func (s *Server) ServeSTDIO(ctx context.Context) error {
    return s.mcp.ServeSTDIO(ctx)
}

func (s *Server) ServeHTTP(ctx context.Context, addr string) error {
    return s.mcp.ServeHTTP(ctx, addr)
}
```

### 5.4 Handler Implementation Pattern

```go
func (s *Server) handleGetAircraftPosition(ctx context.Context, req mcpsdk.CallToolRequest) (mcpsdk.CallToolResult, error) {
    // Extract parameters
    includeAttitude := true
    if v, ok := req.Arguments["include_attitude"].(bool); ok {
        includeAttitude = v
    }

    // Get cached state
    pos, err := s.stateManager.GetPosition(ctx)
    if err != nil {
        return mcpsdk.CallToolResult{}, fmt.Errorf("getting position: %w", err)
    }

    // Build response
    response := AircraftPosition{
        Latitude:       pos.Latitude,
        Longitude:      pos.Longitude,
        AltitudeMSL:    pos.AltitudeMSL,
        AltitudeAGL:    pos.AltitudeAGL,
        HeadingTrue:    pos.HeadingTrue,
        HeadingMag:     pos.HeadingMag,
        IndicatedSpeed: pos.IAS,
        TrueSpeed:      pos.TAS,
        GroundSpeed:    pos.GS,
        VerticalSpeed:  pos.VS,
        Timestamp:      time.Now().UTC().Format(time.RFC3339),
    }

    if includeAttitude {
        response.Pitch = &pos.Pitch
        response.Bank = &pos.Bank
    }

    // Marshal to JSON
    data, err := json.Marshal(response)
    if err != nil {
        return mcpsdk.CallToolResult{}, fmt.Errorf("marshaling response: %w", err)
    }

    return mcpsdk.CallToolResult{
        Content: []mcpsdk.Content{
            {Type: "text", Text: string(data)},
        },
    }, nil
}
```

### 5.5 Error Handling & Graceful Degradation

When SimConnect is unavailable, tools must return **structured errors** that the LLM can understand and act upon.

#### Simulator Error Type

```go
// SimulatorError provides structured error information for LLM consumption
type SimulatorError struct {
    Err         error  // Underlying error
    Message     string // Human-readable message for LLM
    Recoverable bool   // Whether the condition may resolve itself
}

var (
    ErrSimulatorNotConnected = errors.New("simconnect: not connected")
    ErrDataStale             = errors.New("simconnect: data is stale")
)
```

#### Error Response Format

When a tool cannot fulfill a request due to SimConnect unavailability, return a **successful MCP response** with error content (not an MCP protocol error). This allows the LLM to understand and react appropriately.

```go
// SimulatorUnavailableResponse is returned when the simulator is not connected
type SimulatorUnavailableResponse struct {
    Available   bool   `json:"available"`
    Error       string `json:"error"`
    Code        string `json:"code"`
    Recoverable bool   `json:"recoverable"`
    Suggestion  string `json:"suggestion"`
    Timestamp   string `json:"timestamp"`
}
```

**Example error response:**

```json
{
    "available": false,
    "error": "Flight simulator is not running or not connected",
    "code": "SIMULATOR_NOT_CONNECTED",
    "recoverable": true,
    "suggestion": "Start Microsoft Flight Simulator 2024 and load a flight. The MCP server will automatically reconnect.",
    "timestamp": "2024-01-15T10:30:00Z"
}
```

#### Handler Error Pattern

```go
func (s *Server) handleGetAircraftPosition(ctx context.Context, req mcpsdk.CallToolRequest) (mcpsdk.CallToolResult, error) {
    pos, err := s.stateManager.GetPosition(ctx)
    if err != nil {
        // Return structured error, not MCP protocol error
        return s.handleStateError(err)
    }
    // ... build successful response
}

func (s *Server) handleStateError(err error) (mcpsdk.CallToolResult, error) {
    var simErr *SimulatorError
    if errors.As(err, &simErr) {
        response := SimulatorUnavailableResponse{
            Available:   false,
            Error:       simErr.Message,
            Code:        errorCode(simErr.Err),
            Recoverable: simErr.Recoverable,
            Suggestion:  getSuggestion(simErr.Err),
            Timestamp:   time.Now().UTC().Format(time.RFC3339),
        }
        data, _ := json.Marshal(response)

        return mcpsdk.CallToolResult{
            Content: []mcpsdk.Content{{Type: "text", Text: string(data)}},
            IsError: true, // Marks as error but LLM can still parse content
        }, nil
    }
    return mcpsdk.CallToolResult{}, err
}
```

#### Error Codes

| Code | Meaning | Recoverable |
|------|---------|-------------|
| `SIMULATOR_NOT_CONNECTED` | MSFS not running | Yes - will auto-reconnect |
| `DATA_STALE` | Data older than 5 seconds | Yes - wait and retry |
| `SHUTTING_DOWN` | Server is shutting down | No |

### 5.6 Graceful Shutdown

When Kubernetes sends SIGTERM (during scale-down, deployment update, or node drain), the MCP server must shut down gracefully without erroring to the LLM client.

#### Shutdown Sequence

```
1. SIGTERM received
2. Mark as not ready (readiness probe fails)
3. Wait for drain delay (3s) - K8s updates endpoints
4. Stop accepting new MCP requests
5. Wait for in-flight requests to complete (with timeout)
6. Stop state polling goroutines
7. Close SimConnect connection
8. Exit cleanly
```

#### Shutdown Timeouts

```go
const (
    // ShutdownTimeout should be < K8s terminationGracePeriodSeconds (30s)
    ShutdownTimeout = 25 * time.Second

    // DrainDelay allows K8s to update endpoints
    DrainDelay = 3 * time.Second
)
```

#### Signal Handling

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    app, err := NewApplication(ctx)
    if err != nil {
        log.Fatal().Err(err).Msg("failed to create application")
    }

    // Setup signal handling
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

    // Start application
    errChan := make(chan error, 1)
    go func() {
        errChan <- app.Run(ctx)
    }()

    // Wait for shutdown signal
    select {
    case sig := <-sigChan:
        log.Info().Str("signal", sig.String()).Msg("shutdown signal received")
    case err := <-errChan:
        if err != nil && !errors.Is(err, context.Canceled) {
            log.Error().Err(err).Msg("application error")
        }
    }

    // Graceful shutdown
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), ShutdownTimeout)
    defer shutdownCancel()

    if err := app.Shutdown(shutdownCtx); err != nil {
        log.Error().Err(err).Msg("shutdown error")
        os.Exit(1)
    }

    log.Info().Msg("shutdown complete")
}
```

#### Application Shutdown Method

```go
func (a *Application) Shutdown(ctx context.Context) error {
    log.Info().Msg("starting graceful shutdown")

    // Phase 1: Mark as not ready
    a.isReady.Store(false)
    log.Info().Dur("delay", DrainDelay).Msg("waiting for traffic drain")
    time.Sleep(DrainDelay)

    // Phase 2: Stop HTTP server
    if err := a.httpServer.Shutdown(ctx); err != nil {
        log.Error().Err(err).Msg("HTTP server shutdown error")
    }

    // Phase 3: Stop MCP server (wait for in-flight requests)
    if err := a.mcpServer.Shutdown(ctx); err != nil {
        log.Error().Err(err).Msg("MCP server shutdown error")
    }

    // Phase 4: Stop state manager
    if err := a.stateManager.Shutdown(ctx); err != nil {
        log.Error().Err(err).Msg("state manager shutdown error")
    }

    // Phase 5: Close SimConnect
    if err := a.simconnect.Close(); err != nil {
        log.Error().Err(err).Msg("SimConnect close error")
    }

    log.Info().Msg("all components stopped")
    return nil
}
```

#### MCP Server Request Tracking

```go
type Server struct {
    // ... other fields
    activeRequests sync.WaitGroup
    isShutdown     atomic.Bool
}

func (s *Server) Shutdown(ctx context.Context) error {
    s.isShutdown.Store(true)

    // Wait for active requests to complete
    done := make(chan struct{})
    go func() {
        s.activeRequests.Wait()
        close(done)
    }()

    select {
    case <-done:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

// wrapHandler tracks active requests and rejects new ones during shutdown
func (s *Server) wrapHandler(handler ToolHandler) ToolHandler {
    return func(ctx context.Context, req CallToolRequest) (CallToolResult, error) {
        if s.isShutdown.Load() {
            return CallToolResult{}, ErrServerShuttingDown
        }

        s.activeRequests.Add(1)
        defer s.activeRequests.Done()

        return handler(ctx, req)
    }
}
```

---

## 6. Coding Standards

### 6.1 Naming Conventions

```go
// Packages: lowercase, single word preferred
package simconnect
package state

// Exported types: PascalCase
type SimConnectClient struct {}
type AircraftPosition struct {}

// Unexported types: camelCase
type messageHeader struct {}

// Constants: PascalCase for exported, camelCase for internal
const DefaultPort = 4500
const maxRetries = 3

// Interfaces: verb-er pattern or descriptive
type SimConnector interface {
    Connect(ctx context.Context) error
    Close() error
}

type StateReader interface {
    GetPosition(ctx context.Context) (Position, error)
}
```

### 6.2 Error Handling

```go
// Define domain-specific errors
var (
    ErrNotConnected     = errors.New("simconnect: not connected")
    ErrTimeout          = errors.New("simconnect: operation timed out")
    ErrInvalidSimVar    = errors.New("simconnect: invalid simvar")
)

// Wrap errors with context
func (c *Client) RequestData(simvar string) ([]byte, error) {
    if !c.connected {
        return nil, ErrNotConnected
    }

    data, err := c.sendRequest(simvar)
    if err != nil {
        return nil, fmt.Errorf("requesting %s: %w", simvar, err)
    }

    return data, nil
}

// Check specific errors
if errors.Is(err, ErrNotConnected) {
    // Handle reconnection
}
```

### 6.3 Concurrency Patterns

```go
// Context-based cancellation
func (m *Manager) Start(ctx context.Context) error {
    g, ctx := errgroup.WithContext(ctx)

    for _, tier := range m.pollTiers {
        tier := tier // capture loop variable
        g.Go(func() error {
            return m.runPoller(ctx, tier)
        })
    }

    return g.Wait()
}

// Thread-safe state access with RWMutex
type Manager struct {
    mu    sync.RWMutex
    state *AircraftState
}

func (m *Manager) GetPosition(ctx context.Context) (Position, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    if m.state == nil {
        return Position{}, ErrNoData
    }

    return m.state.Position, nil
}

func (m *Manager) updatePosition(pos Position) {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.state.Position = pos
    m.state.LastUpdate = time.Now()
}

// Graceful shutdown
func (s *Server) Shutdown(ctx context.Context) error {
    s.logger.Info().Msg("shutting down server")

    // Stop accepting new requests
    close(s.shutdown)

    // Wait for in-flight requests with timeout
    select {
    case <-s.done:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

### 6.4 Interface Design

```go
// Define interfaces where consumed, not where implemented
// internal/mcp/handlers.go
type StateReader interface {
    GetPosition(ctx context.Context) (types.Position, error)
    GetAutopilot(ctx context.Context) (types.Autopilot, error)
    GetEngines(ctx context.Context) ([]types.Engine, error)
    GetSystems(ctx context.Context, systems []string) (types.Systems, error)
}

// internal/state/manager.go implements StateReader
type Manager struct {
    client SimConnector
    // ...
}

// internal/simconnect/client.go
type SimConnector interface {
    Connect(ctx context.Context) error
    Close() error
    RequestSimVar(name string) (interface{}, error)
    Subscribe(vars []string, callback func(map[string]interface{})) error
}
```

---

## 7. Testing Strategy

### 7.1 TDD Workflow (Red-Green-Refactor)

```
1. RED:    Write a failing test that describes desired behavior
2. GREEN:  Write minimal code to make the test pass
3. REFACTOR: Improve code while keeping tests green
```

### 7.2 Test File Organization

```
internal/simconnect/
├── client.go
├── client_test.go          # Unit tests
├── client_integration_test.go  # Integration tests (build tag)
├── mock_test.go            # Test doubles
```

### 7.3 Interface-Based Mocking

```go
// internal/simconnect/mock_test.go
type MockSimConnector struct {
    ConnectFunc     func(ctx context.Context) error
    CloseFunc       func() error
    RequestFunc     func(name string) (interface{}, error)
    connected       bool
}

func (m *MockSimConnector) Connect(ctx context.Context) error {
    if m.ConnectFunc != nil {
        return m.ConnectFunc(ctx)
    }
    m.connected = true
    return nil
}

func (m *MockSimConnector) RequestSimVar(name string) (interface{}, error) {
    if m.RequestFunc != nil {
        return m.RequestFunc(name)
    }
    return nil, ErrNotConnected
}
```

### 7.4 Table-Driven Tests

```go
func TestSimVarParsing(t *testing.T) {
    tests := []struct {
        name     string
        input    []byte
        varType  string
        expected interface{}
        wantErr  bool
    }{
        {
            name:     "parse float64 altitude",
            input:    []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x88, 0xC3, 0x40},
            varType:  "FLOAT64",
            expected: 10000.0,
            wantErr:  false,
        },
        {
            name:     "parse int32 boolean true",
            input:    []byte{0x01, 0x00, 0x00, 0x00},
            varType:  "INT32",
            expected: int32(1),
            wantErr:  false,
        },
        {
            name:     "invalid empty input",
            input:    []byte{},
            varType:  "FLOAT64",
            expected: nil,
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseSimVarValue(tt.input, tt.varType)

            if tt.wantErr {
                require.Error(t, err)
                return
            }

            require.NoError(t, err)
            assert.Equal(t, tt.expected, got)
        })
    }
}
```

### 7.5 Integration Tests with Build Tags

```go
//go:build integration

package simconnect_test

import (
    "context"
    "testing"
    "time"

    "github.com/yourusername/flightsim-mcp/internal/simconnect"
)

func TestRealSimConnectConnection(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }

    client := simconnect.NewClient(simconnect.Config{
        Host: "192.168.10.100",
        Port: 4500,
    })

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    err := client.Connect(ctx)
    require.NoError(t, err)
    defer client.Close()

    // Test actual SimVar request
    lat, err := client.RequestSimVar("PLANE LATITUDE")
    require.NoError(t, err)
    assert.IsType(t, float64(0), lat)
}
```

### 7.6 Coverage Requirements

| Package | Minimum Coverage |
|---------|------------------|
| `internal/simconnect` | 80% |
| `internal/state` | 80% |
| `internal/mcp` | 80% |
| `internal/config` | 75% |
| `pkg/types` | 70% |
| **Overall** | **75%** |

### 7.7 CI Test Commands

```makefile
.PHONY: test test-race test-coverage test-integration

test:
	go test -v ./...

test-race:
	go test -race -v ./...

test-coverage:
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out
	@go tool cover -func=coverage.out | grep total | awk '{print $$3}' | \
		awk -F'%' '{if ($$1 < 75) exit 1}'

test-integration:
	go test -tags=integration -v ./...
```

---

## 8. Security Requirements

### 8.1 READ-ONLY Enforcement

```go
// MVP: NO WRITE OPERATIONS
// This is enforced at the tool registration level

// FORBIDDEN for MVP - do not implement:
// - set_autopilot_heading
// - set_throttle
// - engage_autopilot
// - Any SimConnect write events

// All tools must be read-only queries
```

### 8.2 Input Validation

```go
// SimVar allowlist - only these can be requested
var AllowedSimVars = map[string]bool{
    "PLANE LATITUDE":          true,
    "PLANE LONGITUDE":         true,
    "PLANE ALTITUDE":          true,
    // ... all 70+ explicitly listed
}

func ValidateSimVarRequest(name string) error {
    if !AllowedSimVars[name] {
        return fmt.Errorf("simvar not in allowlist: %s", name)
    }
    return nil
}

// Parameter validation
func ValidateEngineNumber(n int) error {
    if n < 1 || n > 2 {
        return fmt.Errorf("engine number must be 1 or 2, got: %d", n)
    }
    return nil
}

// Systems filter validation
var AllowedSystems = map[string]bool{
    "fuel":           true,
    "electrical":     true,
    "hydraulics":     true,
    "pressurization": true,
    "controls":       true,
}

func ValidateSystemsFilter(systems []string) error {
    for _, s := range systems {
        if !AllowedSystems[s] {
            return fmt.Errorf("unknown system: %s", s)
        }
    }
    return nil
}
```

### 8.3 Secret Management

```go
// NO hardcoded secrets
// Use environment variables exclusively

type Config struct {
    SimConnectHost string `env:"SIMCONNECT_HOST" envDefault:"192.168.10.100"`
    SimConnectPort int    `env:"SIMCONNECT_PORT" envDefault:"4500"`
    LogLevel       string `env:"LOG_LEVEL" envDefault:"info"`
    HTTPAddr       string `env:"HTTP_ADDR" envDefault:":8080"`
}

// Load config from environment
func LoadConfig() (*Config, error) {
    cfg := &Config{}
    if err := env.Parse(cfg); err != nil {
        return nil, fmt.Errorf("parsing env config: %w", err)
    }
    return cfg, nil
}
```

### 8.4 Rate Limiting

```go
// Rate limit MCP tool calls
type RateLimiter struct {
    limiter *rate.Limiter
}

func NewRateLimiter(rps float64, burst int) *RateLimiter {
    return &RateLimiter{
        limiter: rate.NewLimiter(rate.Limit(rps), burst),
    }
}

func (r *RateLimiter) Allow() bool {
    return r.limiter.Allow()
}

// Usage in handlers
func (s *Server) handleGetAircraftPosition(ctx context.Context, req mcpsdk.CallToolRequest) (mcpsdk.CallToolResult, error) {
    if !s.rateLimiter.Allow() {
        return mcpsdk.CallToolResult{}, ErrRateLimited
    }
    // ... handler logic
}
```

### 8.5 Secure Logging

```go
// Automatic redaction of sensitive fields
type SecureLogger struct {
    logger zerolog.Logger
}

var sensitiveFields = []string{"password", "token", "secret", "key", "auth"}

func (l *SecureLogger) Info() *zerolog.Event {
    return l.logger.Info().Hook(redactionHook{})
}

type redactionHook struct{}

func (h redactionHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
    // Fields are automatically checked and redacted
}
```

### 8.6 CI Security Scanning

```yaml
# .github/workflows/security.yml
security:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4

    - name: Run gosec
      uses: securego/gosec@v2
      with:
        args: ./...

    - name: Run govulncheck
      run: |
        go install golang.org/x/vuln/cmd/govulncheck@latest
        govulncheck ./...

    - name: Run trivy on image
      uses: aquasecurity/trivy-action@master
      with:
        image-ref: 'ghcr.io/${{ github.repository }}:${{ github.sha }}'
        severity: 'HIGH,CRITICAL'
```

---

## 9. CI/CD Pipeline

### 9.1 Overview

The CI/CD pipeline uses GitHub Actions to automate testing, security scanning, building, and publishing container images to GitHub Container Registry (ghcr.io). Rancher Fleet monitors the repository and automatically deploys updated manifests to the K3s cluster.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CI/CD FLOW                                      │
│                                                                              │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌────────┐│
│  │  Push/PR │───▶│   Test   │───▶│ Security │───▶│  Build   │───▶│  Push  ││
│  │          │    │  + Race  │    │   Scan   │    │  Image   │    │ ghcr.io││
│  └──────────┘    └──────────┘    └──────────┘    └──────────┘    └────┬───┘│
│                                                                        │    │
│                    ┌───────────────────────────────────────────────────┘    │
│                    │                                                        │
│                    ▼                                                        │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │                    Rancher Fleet (GitOps)                             │  │
│  │  - Monitors deploy/fleet/ directory                                   │  │
│  │  - Detects image tag changes in values.yaml                          │  │
│  │  - Deploys to K3s cluster automatically                              │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 9.2 GitHub Actions Workflows

#### 9.2.1 CI Workflow (on Push/PR)

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

env:
  GO_VERSION: '1.22'

jobs:
  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}

      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest
          args: --timeout=5m

  test:
    name: Test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true

      - name: Run tests with race detection
        run: go test -race -coverprofile=coverage.out -covermode=atomic ./...

      - name: Check coverage threshold
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "Total coverage: ${COVERAGE}%"
          if (( $(echo "$COVERAGE < 75" | bc -l) )); then
            echo "Coverage ${COVERAGE}% is below 75% threshold"
            exit 1
          fi

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v4
        with:
          files: ./coverage.out
          fail_ci_if_error: false

  security:
    name: Security Scan
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true

      - name: Run gosec
        uses: securego/gosec@master
        with:
          args: '-no-fail -fmt json -out gosec-results.json ./...'

      - name: Run govulncheck
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          govulncheck ./...

  build:
    name: Build
    runs-on: ubuntu-latest
    needs: [lint, test, security]
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true

      - name: Build binary
        run: |
          CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
            -ldflags="-w -s -X main.version=${{ github.sha }}" \
            -o flightsim-mcp ./cmd/flightsim-mcp

      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: flightsim-mcp-linux-amd64
          path: flightsim-mcp
```

#### 9.2.2 Release Workflow (on Tag)

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - 'v*.*.*'

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}
  GO_VERSION: '1.22'

jobs:
  test:
    name: Test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true

      - name: Run tests
        run: go test -race -coverprofile=coverage.out ./...

      - name: Check coverage
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          if (( $(echo "$COVERAGE < 75" | bc -l) )); then
            echo "Coverage ${COVERAGE}% below threshold"
            exit 1
          fi

  security:
    name: Security Scan
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}

      - name: Run gosec
        uses: securego/gosec@master
        with:
          args: ./...

      - name: Run govulncheck
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          govulncheck ./...

  build-and-push:
    name: Build and Push Image
    runs-on: ubuntu-latest
    needs: [test, security]
    permissions:
      contents: read
      packages: write
      attestations: write
      id-token: write
    steps:
      - uses: actions/checkout@v4

      - name: Set up QEMU
        uses: docker/setup-qemu-action@v3

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Log in to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          tags: |
            type=semver,pattern={{version}}
            type=semver,pattern={{major}}.{{minor}}
            type=semver,pattern={{major}}
            type=sha,prefix=sha-
            type=raw,value=latest,enable={{is_default_branch}}

      - name: Build and push
        id: push
        uses: docker/build-push-action@v5
        with:
          context: .
          file: ./deploy/docker/Dockerfile
          platforms: linux/amd64,linux/arm64
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
          build-args: |
            VERSION=${{ github.ref_name }}

      - name: Generate artifact attestation
        uses: actions/attest-build-provenance@v1
        with:
          subject-name: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          subject-digest: ${{ steps.push.outputs.digest }}
          push-to-registry: true

  trivy-scan:
    name: Container Security Scan
    runs-on: ubuntu-latest
    needs: [build-and-push]
    steps:
      - name: Run Trivy vulnerability scanner
        uses: aquasecurity/trivy-action@master
        with:
          image-ref: ghcr.io/${{ github.repository }}:${{ github.ref_name }}
          format: 'sarif'
          output: 'trivy-results.sarif'
          severity: 'HIGH,CRITICAL'

      - name: Upload Trivy scan results
        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: 'trivy-results.sarif'

  create-release:
    name: Create GitHub Release
    runs-on: ubuntu-latest
    needs: [build-and-push, trivy-scan]
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4

      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          generate_release_notes: true
          body: |
            ## Container Image

            ```bash
            docker pull ghcr.io/${{ github.repository }}:${{ github.ref_name }}
            ```

            ## Platforms
            - linux/amd64
            - linux/arm64
```

### 9.3 Image Tagging Strategy

| Trigger | Tag(s) Generated | Example |
|---------|------------------|---------|
| Push to main | `sha-<short-sha>` | `sha-a1b2c3d` |
| Tag `v1.2.3` | `1.2.3`, `1.2`, `1`, `latest` | `1.2.3`, `1.2`, `1`, `latest` |
| PR | None (build only, no push) | - |

**Semantic Versioning:**
- **Major** (v2.0.0): Breaking API changes
- **Minor** (v1.1.0): New features, backward compatible
- **Patch** (v1.0.1): Bug fixes

### 9.4 Makefile

```makefile
# Makefile
.PHONY: all build test test-race test-coverage lint security clean docker-build docker-push

# Variables
VERSION ?= $(shell git describe --tags --always --dirty)
IMAGE_NAME ?= ghcr.io/yourusername/flightsim-mcp
GO_FILES := $(shell find . -name '*.go' -type f)

# Default target
all: lint test build

# Build binary
build:
	CGO_ENABLED=0 go build -ldflags="-w -s -X main.version=$(VERSION)" \
		-o bin/flightsim-mcp ./cmd/flightsim-mcp

# Run tests
test:
	go test -v ./...

# Run tests with race detection
test-race:
	go test -race -v ./...

# Run tests with coverage
test-coverage:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out
	@go tool cover -func=coverage.out | grep total | awk '{print $$3}' | \
		awk -F'%' '{if ($$1 < 75) { print "Coverage below 75%"; exit 1 }}'

# Run tests with HTML coverage report
test-coverage-html: test-coverage
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run linter
lint:
	golangci-lint run --timeout=5m ./...

# Run security scans
security: security-gosec security-govulncheck

security-gosec:
	gosec ./...

security-govulncheck:
	govulncheck ./...

# Format code
fmt:
	gofmt -s -w $(GO_FILES)
	goimports -w $(GO_FILES)

# Tidy dependencies
tidy:
	go mod tidy
	go mod verify

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html

# Docker build (local)
docker-build:
	docker build -t $(IMAGE_NAME):$(VERSION) -f deploy/docker/Dockerfile .

# Docker build multi-platform
docker-build-multi:
	docker buildx build --platform linux/amd64,linux/arm64 \
		-t $(IMAGE_NAME):$(VERSION) -f deploy/docker/Dockerfile .

# Docker push (requires authentication)
docker-push:
	docker push $(IMAGE_NAME):$(VERSION)

# Run locally
run:
	go run ./cmd/flightsim-mcp

# Install development tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install golang.org/x/tools/cmd/goimports@latest

# Pre-commit checks (run before committing)
pre-commit: fmt lint test-race security
	@echo "All pre-commit checks passed!"

# CI simulation (mirrors GitHub Actions)
ci: lint test-coverage security build
	@echo "CI simulation passed!"
```

### 9.5 Project Structure Updates

Add these files to the project structure:

```
flightsim-mcp/
├── .github/
│   └── workflows/
│       ├── ci.yml              # CI workflow (push/PR)
│       └── release.yml         # Release workflow (tags)
├── .golangci.yml               # Linter configuration
├── Makefile                    # Build automation
└── ...
```

### 9.6 golangci-lint Configuration

```yaml
# .golangci.yml
run:
  timeout: 5m
  modules-download-mode: readonly

linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gosec
    - bodyclose
    - nilerr
    - nilnil
    - prealloc
    - revive
    - unconvert
    - unparam

linters-settings:
  gosec:
    severity: medium
    confidence: medium
  revive:
    rules:
      - name: exported
        severity: warning
  errcheck:
    check-type-assertions: true
    check-blank: true

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - gosec
        - errcheck
    - path: mock
      linters:
        - unused
```

### 9.7 Release Process

#### Creating a Release

```bash
# 1. Ensure main is up to date
git checkout main
git pull origin main

# 2. Run all checks locally
make ci

# 3. Create and push tag
git tag -a v1.0.0 -m "Release v1.0.0: Initial MVP"
git push origin v1.0.0

# 4. GitHub Actions automatically:
#    - Runs tests and security scans
#    - Builds multi-platform image
#    - Pushes to ghcr.io with tags: 1.0.0, 1.0, 1, latest
#    - Creates GitHub Release with notes
```

#### Updating Deployment

After a release, update the image tag in Fleet values:

```yaml
# deploy/fleet/values.yaml
image:
  repository: ghcr.io/yourusername/flightsim-mcp
  tag: "v1.0.0"  # Update this to new version
```

Fleet automatically detects the change and redeploys.

### 9.8 Integration with Rancher Fleet

Fleet monitors the `deploy/fleet/` directory. When `values.yaml` is updated with a new image tag:

1. Fleet detects the Git change
2. Fleet reconciles the Helm release
3. Kubernetes pulls the new image from ghcr.io
4. Rolling update deploys the new version

**Image Pull Configuration:**

```yaml
# deploy/fleet/values.yaml
image:
  repository: ghcr.io/yourusername/flightsim-mcp
  pullPolicy: IfNotPresent
  tag: "v1.0.0"

imagePullSecrets:
  - name: ghcr-secret  # If repo is private
```

**For Private Repositories:**

```bash
# Create image pull secret (one-time setup)
kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username=YOUR_GITHUB_USERNAME \
  --docker-password=YOUR_GITHUB_PAT \
  --docker-email=YOUR_EMAIL \
  -n flightsim
```

### 9.9 CI/CD Checklist

#### PR Checklist (Automated)
- [ ] Code compiles
- [ ] All tests pass
- [ ] Test coverage ≥ 75%
- [ ] No race conditions detected
- [ ] golangci-lint passes
- [ ] gosec finds no high/critical issues
- [ ] govulncheck finds no vulnerabilities

#### Release Checklist
- [ ] All CI checks pass
- [ ] Version tag follows semver (vX.Y.Z)
- [ ] Multi-platform image built (amd64, arm64)
- [ ] Image pushed to ghcr.io
- [ ] Trivy scan passes (no critical vulnerabilities)
- [ ] GitHub Release created
- [ ] Fleet values.yaml updated with new tag

---

## 10. Deployment

### 10.1 Dockerfile

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build with security flags
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.version=${VERSION}" \
    -o /flightsim-mcp ./cmd/flightsim-mcp

# Runtime stage - distroless for security
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /flightsim-mcp /flightsim-mcp

USER nonroot:nonroot

ENTRYPOINT ["/flightsim-mcp"]
```

### 10.2 Fleet Configuration

```yaml
# deploy/fleet/fleet.yaml
defaultNamespace: flightsim
helm:
  releaseName: flightsim-mcp
  chart: ./
  values:
    image:
      repository: ghcr.io/yourusername/flightsim-mcp
      tag: v1.0.0

    simconnect:
      host: 192.168.10.100  # Windows gaming PC
      port: 4500

    resources:
      requests:
        memory: "64Mi"
        cpu: "100m"
      limits:
        memory: "128Mi"
        cpu: "500m"
```

### 10.3 Helm Chart Values

```yaml
# deploy/fleet/values.yaml
replicaCount: 1

image:
  repository: ghcr.io/yourusername/flightsim-mcp
  pullPolicy: IfNotPresent
  tag: "v1.0.0"

service:
  type: ClusterIP
  port: 8080

ingress:
  enabled: true
  className: traefik
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - host: flightsim-mcp.moria-lab.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: flightsim-mcp-tls
      hosts:
        - flightsim-mcp.moria-lab.com

env:
  - name: SIMCONNECT_HOST
    value: "192.168.10.100"
  - name: SIMCONNECT_PORT
    value: "4500"
  - name: LOG_LEVEL
    value: "info"
  - name: HTTP_ADDR
    value: ":8080"

securityContext:
  runAsNonRoot: true
  runAsUser: 65534
  readOnlyRootFilesystem: true
  capabilities:
    drop:
      - ALL

networkPolicy:
  enabled: true
  egress:
    - to:
        - ipBlock:
            cidr: 192.168.10.100/32  # Windows host only
      ports:
        - port: 4500
          protocol: TCP
```

### 10.4 Kubernetes Manifests

```yaml
# deploy/fleet/templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "flightsim-mcp.fullname" . }}
spec:
  replicas: {{ .Values.replicaCount }}
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 0
      maxSurge: 1
  selector:
    matchLabels:
      app: {{ include "flightsim-mcp.name" . }}
  template:
    metadata:
      labels:
        app: {{ include "flightsim-mcp.name" . }}
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      # Allow 35s for graceful shutdown (app uses 25s + 5s buffer)
      terminationGracePeriodSeconds: 35
      securityContext:
        {{- toYaml .Values.securityContext | nindent 8 }}
      containers:
        - name: {{ .Chart.Name }}
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
          ports:
            - containerPort: 8080
          env:
            {{- toYaml .Values.env | nindent 12 }}
          resources:
            {{- toYaml .Values.resources | nindent 12 }}
          # Lifecycle hooks for graceful shutdown
          lifecycle:
            preStop:
              exec:
                # Give K8s time to remove pod from endpoints
                command: ["sleep", "5"]
          # Liveness: Is the process alive and healthy?
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
            timeoutSeconds: 5
            failureThreshold: 3
          # Readiness: Can the pod serve traffic?
          # Fails during startup and shutdown
          readinessProbe:
            httpGet:
              path: /ready
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 5
            timeoutSeconds: 3
            failureThreshold: 3
            successThreshold: 1
```

#### Probe Behavior

| Probe | Endpoint | Returns 200 | Returns 503 |
|-------|----------|-------------|-------------|
| **Liveness** (`/health`) | Always 200 if process is alive | Process healthy | Process should be restarted |
| **Readiness** (`/ready`) | Ready to serve traffic | App ready | During startup, shutdown, or SimConnect issues |

**Note:** Readiness probe failing does NOT cause pod restart - it only removes the pod from service endpoints.

### 10.5 Network Policy

```yaml
# deploy/fleet/templates/networkpolicy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: {{ include "flightsim-mcp.fullname" . }}
spec:
  podSelector:
    matchLabels:
      app: {{ include "flightsim-mcp.name" . }}
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: traefik
      ports:
        - port: 8080
  egress:
    # SimConnect to Windows host
    - to:
        - ipBlock:
            cidr: 192.168.10.100/32
      ports:
        - port: 4500
          protocol: TCP
    # DNS
    - to:
        - namespaceSelector: {}
          podSelector:
            matchLabels:
              k8s-app: kube-dns
      ports:
        - port: 53
          protocol: UDP
```

---

## 11. Configuration

### 11.1 Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SIMCONNECT_HOST` | `192.168.10.100` | Windows host running MSFS |
| `SIMCONNECT_PORT` | `4500` | SimConnect TCP port |
| `SIMCONNECT_TIMEOUT` | `10s` | Connection timeout |
| `LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |
| `LOG_FORMAT` | `json` | Log format (json, console) |
| `HTTP_ADDR` | `:8080` | HTTP server address |
| `TRANSPORT` | `stdio` | MCP transport (stdio, http) |
| `METRICS_ENABLED` | `true` | Enable Prometheus metrics |

### 11.2 Configuration Loading

```go
// internal/config/config.go
package config

import (
    "time"

    "github.com/caarlos0/env/v10"
)

type Config struct {
    SimConnect SimConnectConfig
    Server     ServerConfig
    Logging    LoggingConfig
}

type SimConnectConfig struct {
    Host    string        `env:"SIMCONNECT_HOST" envDefault:"192.168.10.100"`
    Port    int           `env:"SIMCONNECT_PORT" envDefault:"4500"`
    Timeout time.Duration `env:"SIMCONNECT_TIMEOUT" envDefault:"10s"`
}

type ServerConfig struct {
    HTTPAddr       string `env:"HTTP_ADDR" envDefault:":8080"`
    Transport      string `env:"TRANSPORT" envDefault:"stdio"`
    MetricsEnabled bool   `env:"METRICS_ENABLED" envDefault:"true"`
}

type LoggingConfig struct {
    Level  string `env:"LOG_LEVEL" envDefault:"info"`
    Format string `env:"LOG_FORMAT" envDefault:"json"`
}

func Load() (*Config, error) {
    cfg := &Config{}
    if err := env.Parse(cfg); err != nil {
        return nil, err
    }
    return cfg, nil
}
```

### 11.3 SimConnect.xml Setup on Windows Host

**Location:** `%APPDATA%\Microsoft Flight Simulator\SimConnect.xml`

```xml
<?xml version="1.0" encoding="Windows-1252"?>
<SimBase.Document Type="SimConnect" version="1,0">
    <Filename>SimConnect.xml</Filename>
    <SimConnect.Comm>
        <Protocol>IPv4</Protocol>
        <Scope>global</Scope>
        <Address>0.0.0.0</Address>
        <Port>4500</Port>
        <MaxClients>64</MaxClients>
        <MaxRecvSize>41088</MaxRecvSize>
        <DisableNagle>False</DisableNagle>
    </SimConnect.Comm>
</SimBase.Document>
```

**Windows Firewall Rule:**
```powershell
New-NetFirewallRule -DisplayName "SimConnect TCP" -Direction Inbound -Protocol TCP -LocalPort 4500 -Action Allow
```

---

## 12. Logging & Monitoring

### 12.1 Structured Logging with Zerolog

```go
package main

import (
    "os"
    "time"

    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
)

func setupLogging(cfg *config.LoggingConfig) {
    // Set log level
    level, err := zerolog.ParseLevel(cfg.Level)
    if err != nil {
        level = zerolog.InfoLevel
    }
    zerolog.SetGlobalLevel(level)

    // Configure output format
    if cfg.Format == "console" {
        log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
    } else {
        // JSON format for Loki ingestion
        zerolog.TimeFieldFormat = time.RFC3339Nano
    }

    // Add standard fields
    log.Logger = log.With().
        Str("service", "flightsim-mcp").
        Str("version", Version).
        Logger()
}
```

### 12.2 Log Format for Loki

```json
{
  "level": "info",
  "service": "flightsim-mcp",
  "version": "1.0.0",
  "component": "simconnect",
  "action": "connect",
  "host": "192.168.10.100",
  "port": 4500,
  "latency_ms": 15,
  "time": "2024-01-15T10:30:00.000Z",
  "message": "connected to SimConnect"
}
```

### 12.3 Prometheus Metrics

```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // MCP tool call metrics
    ToolCallsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "flightsim_mcp_tool_calls_total",
            Help: "Total number of MCP tool calls",
        },
        []string{"tool", "status"},
    )

    ToolCallDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "flightsim_mcp_tool_call_duration_seconds",
            Help:    "Duration of MCP tool calls",
            Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
        },
        []string{"tool"},
    )

    // SimConnect metrics
    SimConnectConnected = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "flightsim_simconnect_connected",
            Help: "SimConnect connection status (1=connected, 0=disconnected)",
        },
    )

    SimConnectRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "flightsim_simconnect_requests_total",
            Help: "Total SimConnect requests",
        },
        []string{"type", "status"},
    )

    SimConnectLatency = promauto.NewHistogram(
        prometheus.HistogramOpts{
            Name:    "flightsim_simconnect_latency_seconds",
            Help:    "SimConnect request latency",
            Buckets: []float64{.001, .005, .01, .025, .05, .1},
        },
    )

    // State cache metrics
    StateCacheAge = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "flightsim_state_cache_age_seconds",
            Help: "Age of cached state data",
        },
        []string{"category"},
    )
)
```

### 12.4 Prometheus ServiceMonitor

```yaml
# deploy/fleet/templates/servicemonitor.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: {{ include "flightsim-mcp.fullname" . }}
  labels:
    release: kube-prometheus-stack
spec:
  selector:
    matchLabels:
      app: {{ include "flightsim-mcp.name" . }}
  endpoints:
    - port: http
      path: /metrics
      interval: 15s
```

### 12.5 Grafana Dashboard Queries

```promql
# MCP Tool Call Rate
sum(rate(flightsim_mcp_tool_calls_total[5m])) by (tool)

# Tool Call Latency P99
histogram_quantile(0.99, rate(flightsim_mcp_tool_call_duration_seconds_bucket[5m]))

# SimConnect Connection Status
flightsim_simconnect_connected

# SimConnect Error Rate
sum(rate(flightsim_simconnect_requests_total{status="error"}[5m]))
/ sum(rate(flightsim_simconnect_requests_total[5m]))
```

---

## 13. Implementation Phases

### Phase 1: Proof of Concept (Week 1)

**Goal:** Prove we can connect to SimConnect and read a single SimVar

**Deliverables:**
1. Basic project structure with `go.mod`
2. SimConnect TCP client with connection/disconnection
3. Wire protocol: header encoding, single message exchange
4. Read ONE SimVar: `PLANE LATITUDE`
5. Basic unit tests for protocol encoding
6. Manual test against real MSFS
7. GitHub Actions CI workflow (test, lint, security)
8. Basic Makefile with CI-aligned targets

**Success Criteria:**
- [ ] Can connect to SimConnect over TCP
- [ ] Can request and receive PLANE LATITUDE value
- [ ] Tests pass with mocked connection
- [ ] GitHub Actions CI passes on push

**Files to Create:**
```
.github/workflows/ci.yml
.golangci.yml
cmd/flightsim-mcp/main.go
internal/simconnect/client.go
internal/simconnect/client_test.go
internal/simconnect/protocol.go
internal/simconnect/protocol_test.go
go.mod
go.sum
Makefile
```

### Phase 2: Core SimConnect Client (Week 2)

**Goal:** Complete SimConnect client with all MVP SimVars

**Deliverables:**
1. Full wire protocol implementation
2. SimVar registry with all 70+ variables
3. Data definition setup and subscription
4. Polling tier implementation (fast/medium/slow)
5. Reconnection logic with backoff
6. Comprehensive unit tests

**Success Criteria:**
- [ ] All SimVar categories working
- [ ] Polling tiers running concurrently
- [ ] Automatic reconnection on disconnect
- [ ] 80% test coverage on simconnect package

### Phase 3: State Management (Week 3)

**Goal:** Thread-safe state caching layer

**Deliverables:**
1. State manager with RWMutex protection
2. Polling orchestration
3. Data transformation to typed structs
4. State staleness detection
5. Unit tests with race detection

**Success Criteria:**
- [ ] Concurrent access without data races
- [ ] State freshness guarantees per tier
- [ ] 80% test coverage on state package

### Phase 4: MCP Server Integration (Week 4)

**Goal:** Working MCP server with all tools and graceful error handling

**Deliverables:**
1. MCP server using official Go SDK
2. All five tool handlers implemented (including get_simulator_status)
3. STDIO transport for Claude Desktop
4. Input validation with allowlists
5. Graceful error responses when SimConnect unavailable
6. Graceful shutdown with SIGTERM handling
7. Unit tests for handlers

**Success Criteria:**
- [ ] All tools working via STDIO
- [ ] Valid JSON responses
- [ ] Graceful errors when simulator disconnected
- [ ] Input validation rejects invalid requests
- [ ] 80% test coverage on mcp package

### Phase 5: HTTP Transport & Containerization (Week 5)

**Goal:** Remote access via HTTP and Docker deployment with full CI/CD

**Deliverables:**
1. HTTP/SSE transport implementation
2. Health and readiness endpoints
3. Prometheus metrics endpoint
4. Dockerfile (distroless, non-root)
5. Integration tests with build tags
6. GitHub Actions release workflow (build + push to ghcr.io)
7. Multi-platform builds (amd64, arm64)
8. Container security scanning with Trivy

**Success Criteria:**
- [ ] HTTP transport working
- [ ] Container builds successfully
- [ ] Metrics exposed on /metrics
- [ ] Image passes security scan
- [ ] Release workflow pushes to ghcr.io on tag
- [ ] Trivy scan finds no critical vulnerabilities

### Phase 6: Kubernetes Deployment (Week 6)

**Goal:** Fleet/GitOps deployment to homelab with full CI/CD integration

**Deliverables:**
1. Helm chart with all resources
2. Fleet configuration pointing to ghcr.io image
3. Network policies (restrict egress to SimConnect host)
4. ServiceMonitor for Prometheus
5. Image pull from ghcr.io (with secret if private)
6. End-to-end release process documentation
5. Documentation and runbooks

**Success Criteria:**
- [ ] Deploys via Fleet GitOps
- [ ] Traefik ingress working
- [ ] TLS via cert-manager
- [ ] Metrics visible in Grafana
- [ ] README with setup instructions

---

## 14. CLAUDE.md Template

```markdown
# FlightSim-MCP Project Guide

## Quick Start
- Run tests: `make test`
- Run tests with race detection: `make test-race`
- Build: `make build`
- Run locally: `./bin/flightsim-mcp`
- Docker build: `make docker-build`
- Full CI simulation: `make ci`
- Pre-commit checks: `make pre-commit`

## Architecture
See `spec.md` for full architecture. Key components:
- `internal/simconnect/` - SimConnect wire protocol client
- `internal/state/` - Thread-safe state caching
- `internal/mcp/` - MCP server and tool handlers
- `internal/transport/` - STDIO and HTTP transports

## Coding Standards
- Follow Go conventions (gofmt, golint)
- Interfaces defined where consumed, not implemented
- Error wrapping with `fmt.Errorf("context: %w", err)`
- Context-based cancellation for all I/O
- Table-driven tests for all business logic

## Testing
- TDD: Write test first, then implementation
- Run with race detection: `make test-race`
- Coverage target: 75% overall, 80% core packages
- Integration tests: `make test-integration` (requires MSFS)

## SimConnect Protocol
- TCP connection to Windows host port 4500
- Binary protocol with 16-byte headers
- Little-endian byte order
- See `docs/simvars.md` for complete SimVar reference

## MVP Constraints
- **READ-ONLY**: No write operations to simulator
- **A320 Only**: CFM engine SimVars
- **5 Tools**: get_simulator_status, get_aircraft_position, get_autopilot_state, get_engine_status, get_systems_status

## Security
- No hardcoded secrets (use env vars)
- Input validation with allowlists
- Rate limiting on tool calls
- Run gosec before commits: `make security`

## CI/CD
- GitHub Actions workflows in `.github/workflows/`
- CI runs on push/PR: lint, test, security scan, build
- Release on tag (vX.Y.Z): build + push to ghcr.io
- Fleet GitOps deploys from `deploy/fleet/`
- Image registry: ghcr.io/yourusername/flightsim-mcp

## Releases
- Create tag: `git tag -a v1.0.0 -m "Release v1.0.0"`
- Push tag: `git push origin v1.0.0`
- Update `deploy/fleet/values.yaml` with new tag

## Git Commits
- 2-3 sentences max, concise and clear
- Format: "Add X" / "Fix Y" / "Update Z"

## Key Files
- `spec.md` - Complete specification
- `.github/workflows/` - CI/CD workflows
- `internal/simconnect/simvars.go` - SimVar definitions
- `internal/mcp/tools.go` - MCP tool definitions
- `deploy/fleet/values.yaml` - Kubernetes configuration
```

---

## 15. Starter Prompt

Use this prompt when starting implementation with a new AI coding agent:

---

**Starter Prompt for AI Coding Agent:**

```
You are implementing FlightSim-MCP, a Go-based MCP server that connects to Microsoft Flight Simulator 2024 via SimConnect and exposes aircraft state data to LLM applications.

Read the specification at `/Users/eythandecker/dev/flightsim-mcp/spec.md` thoroughly before starting.

KEY CONSTRAINTS:
1. READ-ONLY operations only (no simulator write operations)
2. A320 aircraft with CFM engines
3. TCP connection to SimConnect (no Go library exists - implement wire protocol)
4. Follow TDD: Write failing test first, then implementation

START WITH PHASE 1:
1. Create basic project structure with go.mod
2. Implement SimConnect TCP client that can connect/disconnect
3. Implement wire protocol header encoding (16 bytes, little-endian)
4. Read ONE SimVar: PLANE LATITUDE
5. Write unit tests for protocol encoding

Use the MCP SDK: github.com/modelcontextprotocol/go-sdk

CRITICAL IMPLEMENTATION NOTES:
- SimConnect uses binary protocol with 16-byte headers
- Message format: Size(4) | Version(4) | Type(4) | ID(4) | Payload(variable)
- All integers are little-endian
- SimConnect version for MSFS 2024 is 4

Begin by creating an implementation plan you will update at the root of this repo to keep track of progress on the implementation of this spec over the course of many session. Afterwards, begin imlementing the project structure, setting up Claude.md and implementing the connection logic. Ask any clairfying questions you have before getting started. For now, you will only be able to do unit tests and no integration tests until i get a flight sim instance up and running.
```

---

## Appendix A: Known Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| No mature Go SimConnect library | High | Implement wire protocol from scratch using MSFS SDK docs |
| Wire protocol undocumented details | Medium | Use Wireshark to capture real SimConnect traffic |
| SimConnect version differences | Medium | Target MSFS 2024 specifically, version 4 protocol |
| Network latency to Windows host | Low | Polling tiers reduce real-time requirements |
| Container can't reach Windows host | Medium | Verify network path, use host networking if needed |

## Appendix B: References

- [MSFS SDK SimConnect Documentation](https://docs.flightsimulator.com/html/Programming_Tools/SimConnect/SimConnect_SDK.htm)
- [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk)
- [SimConnect Variable Reference](https://docs.flightsimulator.com/html/Programming_Tools/SimVars/Simulation_Variables.htm)
- [Fleet GitOps Documentation](https://fleet.rancher.io/)
- [Zerolog Logger](https://github.com/rs/zerolog)

---

*This specification was generated for AI coding agent consumption. It prioritizes clarity, completeness, and implementability over prose quality.*

---

## Implementation Progress

### Phase 1: SimConnect Foundation (READ-ONLY Proof of Concept)

- [x] Project scaffolding (go.mod, Makefile, .golangci.yml, directory structure)
- [x] CLAUDE.md with project conventions and SDK API notes
- [x] Wire protocol header encoding/decoding (16-byte little-endian, TDD)
- [x] Sentinel error types (ErrNotConnected, ErrTimeout, ErrInvalidSimVar, ErrConnectionRefused)
- [x] Shared types (SimulatorError, Position)
- [x] SimConnect TCP client (connect, close, sendMessage, readMessage, TDD with net.Pipe)
- [x] SimVar definitions (PlaneLatitude, SimVarRegistry with allowlist validation)
- [x] Binary value parsing (float64, int32 from little-endian bytes, TDD)
- [x] Data definition and request message construction (AddToDataDefinition, RequestData)
- [x] GitHub Actions CI workflow (lint → test → security → build)

### Phase 2: MCP Server + State Management (TODO)

- [ ] State manager with concurrent-safe cache
- [ ] MCP server with get_aircraft_position tool
- [ ] STDIO and HTTP transport
- [ ] Configuration loading
- [ ] Structured logging with zerolog

### Phase 3: Full A320 SimVar Set (TODO)

- [ ] Complete ~70 SimVar definitions across 9 categories
- [ ] Category-based MCP tools
- [ ] Docker deployment with SimConnect.xml configuration
- [ ] Fleet GitOps manifests
