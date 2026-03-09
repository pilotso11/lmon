# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Test Commands

**Building:**
```bash
CGO_ENABLED=0 go build                    # Build the lmon binary
CGO_ENABLED=0 go build -v ./...          # Build all packages with verbose output
```

**Testing:**
```bash
CGO_ENABLED=0 go test -race ./...                               # Run all tests
CGO_ENABLED=0 go test -race -v ./...                           # Run tests with verbose output
CGO_ENABLED=0 go test -race ./...                        # Run tests with race detection
CGO_ENABLED=0 go test -v -race ./... -covermode=atomic -coverprofile=coverage.out  # Full test suite with coverage
go tool cover -func=coverage.out          # View coverage report
```

**UI Tests (requires running web server):**
```bash
CGO_ENABLED=0 go test -v ./uitest         # Run UI tests using go-rod browser automation
```

**Docker:**
```bash
docker build -t lmon .                     # Build Docker image
docker run -p 8080:8080 lmon              # Run in container (node mode)
LMON_MODE=aggregator docker run -p 8080:8080 lmon  # Run in aggregator mode
```

**MCP Integrations:**
This repository supports MCP (Model Context Protocol) integrations for enhanced development workflows:
- GitHub integration for repository management
- browser-tools-mcp for web-based testing and UI validation
- container-use for Docker container management and testing

## Architecture Overview

**lmon** is a lightweight monitoring service written in Go that provides system monitoring with a web dashboard. It supports two operating modes: **node** (standalone agent) and **aggregator** (cluster-wide dashboard).

### Core Components

1. **main.go** - Application entry point that:
   - Loads configuration using Viper
   - Resolves operating mode via `LMON_MODE` env var (node or aggregator)
   - Dispatches to `startNode()` or `startAggregator()` accordingly
   - Handles graceful shutdown

2. **config/** - Configuration management:
   - Uses Viper for YAML config files and environment variables
   - Supports `config.yaml` in current directory or `/etc/lmon/config.yaml`
   - Environment variables prefixed with `LMON_`
   - Configuration can be updated via web UI
   - Mode resolution via `config.ResolveMode()` reading `LMON_MODE`
   - New config sections: `KubernetesConfig`, `AggregatorConfig`, `DatabaseConfig`
   - K8s monitor configs: `K8sEventsConfig`, `K8sNodesConfig`, `K8sServiceConfig`
   - Defaults are applied post-unmarshal (not via viper `SetDefault`) to avoid polluting saved config files

3. **monitors/** - Core monitoring system:
   - **monitors.go** - Service that manages monitor lifecycles and RAG (Red/Amber/Green) status
   - **disk/** - Disk usage monitoring for filesystem paths
   - **system/** - CPU and memory monitoring using gopsutil
   - **healthcheck/** - HTTP endpoint health checks
   - **ping/** - ICMP ping monitoring using pro-bing library
   - **docker/** - Docker container restart count monitoring
   - **k8sevents/** - Kubernetes failure event monitoring (CrashLoopBackOff, OOMKilled, etc.)
   - **k8snodes/** - Kubernetes node condition monitoring (Ready, pressure, cordoned)
   - **k8sservice/** - Kubernetes service pod health monitoring via HTTP probes
   - **mapper/** - Maps configuration to monitor instances using factory methods

4. **aggregator/** - Cluster-wide node aggregation:
   - Discovers lmon node agents via Kubernetes pod labels
   - Scrapes `/metrics` JSON endpoints from each node in parallel
   - Stores results in a thread-safe `xsync.Map`
   - Fires webhooks on node availability state changes
   - Provider interface with K8sProvider (real) and MockProvider (tests)
   - Types: `ScrapedMetric`, `ScrapedPayload`, `NodeResult`, `NodeEndpoint`

5. **db/** - Database layer (PostgreSQL):
   - **db.go** - `MonitorSnapshot` model, `MonitorSummary` struct, `Store` interface
   - **postgres.go** - `PostgresStore` with GORM, non-fatal connection, atomic availability
   - **noop.go** - `NoopStore` for when no database is configured
   - **writer.go** - `BufferedWriter` with non-blocking channel, rate limiting, silent drop
   - **retention.go** - `RetentionManager` with scheduled purging of old snapshots
   - Database is always optional; all methods guard on availability

6. **web/** - HTTP server and UI:
   - Embedded static assets (HTML, CSS, JS) using Go embed
   - REST API endpoints for configuration, status, history, and metrics
   - Template-based web dashboard with node mode and aggregator mode views
   - Sparkline SVG generation for historical status visualization
   - History page (`/history`) with detailed metric drillthrough
   - Aggregator view with per-node status grids
   - Mobile-responsive UI

7. **webhook/** - Notification system:
   - Sends JSON payloads to configured webhook URLs
   - Triggered when monitors change to unhealthy states

8. **deploy/kubernetes/** - Kubernetes manifests:
   - RBAC (ServiceAccounts, ClusterRoles, ClusterRoleBindings)
   - DaemonSet for node agents (hostPID, host filesystem mounts)
   - Deployment for aggregator (single replica)
   - Services (ClusterIP for aggregator, headless for node discovery)
   - ConfigMaps for node and aggregator configurations

### Key Design Patterns

- **Interface-based monitors**: All monitors implement a common `Monitor` interface
- **Provider pattern**: Every monitor type has a Provider interface, DefaultProvider (real), and MockProvider (tests)
- **Concurrent-safe**: Uses `xsync.MapOf` for thread-safe monitor storage
- **Context-aware**: Proper context propagation for graceful shutdown
- **Mock support**: Each monitor has mock implementations for testing
- **Configuration persistence**: Web UI changes are saved back to config files
- **Mode dispatch**: `LMON_MODE` env var controls startup; single binary, no build tags
- **K8s gated at runtime**: All k8s code gated behind `kubernetes.enabled`; when false (default), no k8s client created
- **DB always optional**: `BufferedWriter` channel ensures monitoring never blocks on DB; non-blocking startup; 503 on API when unavailable

### Monitor Types and Thresholds

- **Disk**: Monitors filesystem usage percentage, configurable threshold (default 80%)
- **CPU/Memory**: System resource monitoring with percentage thresholds (default 90%)
- **Health Checks**: HTTP endpoint monitoring with timeout configuration
- **Ping**: Network reachability via ICMP with response time thresholds (Green: <100ms, Amber: >=100ms)
- **Docker**: Container restart count monitoring with configurable threshold
- **K8s Events**: Kubernetes failure event counting (Green: 0 events, Amber: < threshold, Red: >= threshold)
- **K8s Nodes**: Node condition monitoring (Green: all Ready, Amber: pressure/cordoned, Red: NotReady)
- **K8s Service**: Pod health percentage (Green: >= threshold%, Amber: > 50%, Red: <= 50%)

### Testing Strategy

- Unit tests for all major components with race detection
- Mock implementations for external dependencies (Provider pattern)
- DB unit tests use SQLite in-memory via GORM (per-test shared-cache databases)
- UI tests using go-rod for browser automation
- Integration tests in `uitest/` directory
- Configuration change persistence testing
- End-end testing both as go tests and via UI tests.
- Coverage target of 90% or higher.
- Common patterns: `assert.Eventually()`, `t.Context()`, `goleak.VerifyNone(t)`, `-race`

### Web UI Structure

- **Dashboard** (`/`) - Main monitoring view with traffic light indicators (node mode)
- **Aggregator View** (`/`) - Cluster-wide node status grid (aggregator mode)
- **History** (`/history`) - Metric history with sparklines (requires database)
- **Configuration** (`/config`) - Add/remove monitors and update settings
- **Mobile** (`/mobile`) - Mobile-optimized dashboard
- **API endpoints** - RESTful API for programmatic access (`/api/items`, `/api/config`, `/api/history`, `/api/summary`, `/metrics`)

### Web UI Design

- Uses go templates for dynamic server-side content rendering
- CSS styling with Bootstrap for responsive design
- JavaScript for interactivity used minimally where necessary.
- SVG sparklines generated server-side (no charting library dependency)

## Implementation Goals
- Keep the implementation simple and lightweight
- The application follows Go best practices with proper error handling, context usage, and concurrent programming patterns.
- Use Go's standard library as much as possible
- Avoid complex dependencies
- Use a single binary for deployment
* Easy extensibilty and supportability with new monitor implementations
* Extensive use of unit tests, integration tests, and end-to-end tests with a goal of 90% test coverage, and the only gaps being hard to produce exception cases.
* Use of testing libraries like `testify` for assertions and mocking. Attention payed to timeouts using assert.Eventually(), as well as asyc issues, test -race is used for validation.
* Full testing of the UI using rod.
* Rod tests avoid the use of Must functions that panic on failure, instead using error handling to allow for graceful test failures.
* New implementations follow the same patterns and practices as existing implementations.
