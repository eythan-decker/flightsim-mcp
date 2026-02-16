# FlightSim-MCP

## Project Overview
Go-based MCP server connecting to MSFS 2024 via SimConnect binary TCP protocol. Exposes read-only aircraft state data to LLM applications.

## Architecture
- `cmd/flightsim-mcp/` — Entry point
- `internal/simconnect/` — Wire protocol, TCP client, SimVar definitions
- `internal/state/` — Concurrent-safe state cache
- `internal/mcp/` — MCP server tools and handlers
- `internal/transport/` — STDIO/HTTP transport
- `internal/config/` — Configuration loading
- `pkg/types/` — Shared types (aircraft state, errors)

## Build & Test
```bash
make build        # Build binary
make test         # Run tests with race detection
make lint         # Run golangci-lint
make coverage     # Generate coverage report
make all          # fmt + vet + lint + test + build
```

## Code Standards
- **TDD**: Write tests first, then implementation
- **Table-driven tests** with `testify/assert` and `testify/require`
- **Error handling**: Sentinel errors with `errors.Is()`, wrap with `fmt.Errorf("context: %w", err)`
- **Concurrency**: Use `sync.RWMutex` for shared state, `atomic` for simple counters
- **No panics** in library code; return errors instead

## MCP SDK API (v1.3.0)
```go
server := mcp.NewServer(&mcp.Implementation{Name: "flightsim-mcp", Version: "1.0.0"}, nil)
mcp.AddTool(server, &mcp.Tool{Name: "tool_name", Description: "..."}, handlerFunc)
server.Run(ctx, &mcp.StdioTransport{})
```

## SimConnect Wire Protocol
- Little-endian encoding, 16-byte header: Size(4) + Version(4) + Type(4) + ID(4)
- Protocol version: 4
- TCP connection to SimConnect server (default port 4500)

## Git Conventions
- Commits: 2-3 sentences max, concise and clear
- No "authored by Claude Code" in commits
- Atomic commits: one logical change per commit
