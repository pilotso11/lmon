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
docker run -p 8080:8080 lmon              # Run in container
```

**MCP Integrations:**
This repository supports MCP (Model Context Protocol) integrations for enhanced development workflows:
- GitHub integration for repository management
- browser-tools-mcp for web-based testing and UI validation
- container-use for Docker container management and testing

## Architecture Overview

**lmon** is a lightweight monitoring service written in Go that provides system monitoring with a web dashboard.

### Core Components

1. **main.go** - Application entry point that:
   - Loads configuration using Viper
   - Initializes the monitoring service
   - Starts the web server
   - Handles graceful shutdown

2. **config/** - Configuration management:
   - Uses Viper for YAML config files and environment variables
   - Supports `config.yaml` in current directory or `/etc/lmon/config.yaml`
   - Environment variables prefixed with `LMON_`
   - Configuration can be updated via web UI

3. **monitors/** - Core monitoring system:
   - **monitors.go** - Service that manages monitor lifecycles and RAG (Red/Amber/Green) status
   - **disk/** - Disk usage monitoring for filesystem paths
   - **system/** - CPU and memory monitoring using gopsutil
   - **healthcheck/** - HTTP endpoint health checks
   - **ping/** - ICMP ping monitoring using pro-bing library
   - **mapper/** - Maps configuration to monitor instances

4. **web/** - HTTP server and UI:
   - Embedded static assets (HTML, CSS, JS) using Go embed
   - REST API endpoints for configuration and status
   - Template-based web dashboard
   - Mobile-responsive UI
   - Real-time status updates

5. **webhook/** - Notification system:
   - Sends JSON payloads to configured webhook URLs
   - Triggered when monitors change to unhealthy states

### Key Design Patterns

- **Interface-based monitors**: All monitors implement a common `Monitor` interface
- **Concurrent-safe**: Uses `xsync.MapOf` for thread-safe monitor storage
- **Context-aware**: Proper context propagation for graceful shutdown
- **Mock support**: Each monitor has mock implementations for testing
- **Configuration persistence**: Web UI changes are saved back to config files

### Monitor Types and Thresholds

- **Disk**: Monitors filesystem usage percentage, configurable threshold (default 80%)
- **CPU/Memory**: System resource monitoring with percentage thresholds (default 90%)
- **Health Checks**: HTTP endpoint monitoring with timeout configuration
- **Ping**: Network reachability via ICMP with response time thresholds (Green: <100ms, Amber: ≥100ms)

### Testing Strategy

- Unit tests for all major components with race detection
- Mock implementations for external dependencies
- UI tests using go-rod for browser automation
- Integration tests in `uitest/` directory
- Configuration change persistence testing
- End-end testing both as go tests and via UI tests.
- Coverage target of 90% or higher.

### Web UI Structure

- **Dashboard** (`/`) - Main monitoring view with traffic light indicators
- **Configuration** (`/config`) - Add/remove monitors and update settings
- **Mobile** (`/mobile`) - Mobile-optimized dashboard
- **API endpoints** - RESTful API for programmatic access

### Web UI Design

- Uses go templates for dynamic server-side content rendering
- CSS styling with Bootstrap for responsive design
- JavaScript for interactivity used minimally where necessary.

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
