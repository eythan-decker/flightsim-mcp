# FlightSim-MCP: K8s Deployment Project Status

**Goal:** Deploy flightsim-mcp to homelab K8s cluster via Fleet GitOps, with HTTP MCP transport, CI/CD image pipeline, and proper containerization.

**SimConnect Target:** `192.168.0.44:4500`
**Container Registry:** `ghcr.io/eythandecker/flightsim-mcp` (public)
**Homelab Repo:** `../homelab/` — Fleet GitOps, Traefik IngressRoute, wildcard `*.moria-lab.com` TLS

---

## Workstream 1: Containerization
**Status:** Complete

- [x] Create multi-stage Dockerfile (`golang:1.24-alpine` build → `distroless/static:nonroot` runtime)
- [x] Add `.dockerignore`
- [ ] Verify `make docker-build` works locally
- [ ] Validate binary runs correctly in container

---

## Workstream 2: CI/CD Image Pipeline
**Status:** Complete (workflow created, untested)

- [x] Create GitHub Actions release workflow (`.github/workflows/release.yml`)
  - Trigger on tag push (`v*.*.*`)
  - Run tests + security scan
  - Build multi-platform image (amd64 + arm64)
  - Push to `ghcr.io/eythan-decker/flightsim-mcp`
  - Create GitHub Release
- [x] Add Makefile `docker-push` target

---

## Workstream 3: HTTP MCP Transport
**Status:** Complete

- [x] Add `MCPConfig` to config with `MCP_TRANSPORT` and `MCP_HTTP_ADDR` env vars
- [x] Implement HTTP transport using MCP SDK `StreamableHTTPHandler`
- [x] Add health endpoint (`/health`) — liveness, always 200
- [x] Add readiness endpoint (`/ready`) — 503 when stale/no data
- [x] Support dual transport mode: STDIO (default) + HTTP (via `MCP_TRANSPORT=http`)
- [x] Tests for config, HTTP handler, health, and readiness
- [x] Graceful shutdown for HTTP server on SIGTERM

---

## Workstream 4: Fleet Manifests (Homelab Repo)
**Status:** Complete (manifests created, not deployed)

- [x] Create `charts/flightsim-mcp/fleet.yaml`
- [x] Create `deployment.yaml` — single replica, resource limits, env vars, probes
- [x] Create `service.yaml` — ClusterIP + Traefik IngressRoute
- [x] SimConnect env: `SIMCONNECT_HOST=192.168.0.44`, `SIMCONNECT_PORT=4500`
- [x] Ingress: `flightsim-mcp.moria-lab.com` via Traefik websecure

---

## Remaining Verification Steps

- [ ] `make docker-build VERSION=test` succeeds locally
- [ ] `docker run -e MCP_TRANSPORT=http -p 8080:8080 <image>` → `/health` returns 200
- [ ] Local STDIO still works: `./bin/flightsim-mcp` with Claude Code
- [ ] Tag `v1.0.0` → release workflow pushes to GHCR
- [ ] Push homelab manifests → pod comes up
- [ ] With MSFS at 192.168.0.44, readiness flips to 200

---

## Workstream 6: Observability (Future)
**Status:** Deferred

- [ ] Prometheus metrics endpoint (`/metrics`)
- [ ] ServiceMonitor for Prometheus scraping
- [ ] Grafana dashboard
- [ ] Structured logging with zerolog
- [ ] Add to Homarr dashboard

---

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-03-22 | HTTP transport (Option A) | Proper K8s deployment with health probes, ingress, future metrics |
| 2026-03-22 | Public GHCR registry | No pull secrets needed on cluster |
| 2026-03-22 | No node affinity | Can run on any node |
| 2026-03-22 | Always-on pod | App already handles sim offline gracefully with backoff reconnect |
| 2026-03-22 | Observability deferred | Focus on core deployment first |
| 2026-03-23 | Stateless StreamableHTTP | No sticky sessions needed, correct for K8s load balancing |
