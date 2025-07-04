# lmon - Lightweight Monitoring Service

A lightweight monitoring service written in Go that monitors system resources, disk space, and application health.

[![Go](https://github.com/yourusername/lmon/actions/workflows/go.yml/badge.svg)](https://github.com/yourusername/lmon/actions/workflows/go.yml)
[![Docker](https://github.com/yourusername/lmon/actions/workflows/docker.yml/badge.svg)](https://github.com/yourusername/lmon/actions/workflows/docker.yml)
![Coverage](https://img.shields.io/badge/Coverage-0%25-red.svg)

## Features

- Monitor available disk space on specified volumes
- Monitor CPU and memory usage
- Monitor applications via health check endpoints
- Configurable thresholds for all monitored items
- Web UI with Bootstrap showing status as traffic lights
- Webhook notifications for unhealthy states
- Configuration management through the web UI
- Icon support for monitored items

## Requirements

- Go 1.24 or later
- For system monitoring: Linux, macOS, or Windows

## Installation

### From Source

```bash
git clone https://github.com/yourusername/lmon.git
cd lmon
go build
```

### Using Docker

```bash
docker build -t lmon .
docker run -p 8080:8080 -v /path/to/config:/etc/lmon lmon
```

## Configuration

lmon uses a YAML configuration file and environment variables for configuration. The default configuration file is `config.yaml` in the current directory, but it also looks for configuration in `/etc/lmon/config.yaml`.

### Example Configuration

```yaml
web:
  host: "0.0.0.0"
  port: 8080

monitoring:
  interval: 60
  disk:
    - path: "/"
      threshold: 80
      icon: "storage"
  system:
    cpu_threshold: 80
    memory_threshold: 80
    cpu_icon: "speed"
    memory_icon: "memory"
  healthchecks:
    - name: "Example API"
      url: "https://api.example.com/health"
      interval: 60
      timeout: 10
      icon: "cloud"

webhook:
  enabled: false
  url: ""
```

### Environment Variables

All configuration options can be set using environment variables with the prefix `LMON_`. For example:

- `LMON_WEB_HOST`: Web server host
- `LMON_WEB_PORT`: Web server port
- `LMON_MONITORING_INTERVAL`: Monitoring interval in seconds
- `LMON_WEBHOOK_ENABLED`: Enable webhook notifications
- `LMON_WEBHOOK_URL`: Webhook URL

## Running

### Command Line

```bash
./lmon
```

### As a Systemd Service

#### Automatic Installation

lmon provides built-in commands to install and uninstall the systemd service:

```bash
# Install the service (requires root privileges)
sudo ./lmon --install-service

# Uninstall the service (requires root privileges)
sudo ./lmon --uninstall-service
```

The install command will:
1. Copy the binary to `/opt/lmon/lmon`
2. Copy the systemd service file to `/etc/systemd/system/lmon.service`
3. Create a user and group for the service (`lmon`)
4. Create the configuration directory at `/etc/lmon`
5. Enable the service

After installation, you'll need to:
1. Copy your configuration file to `/etc/lmon/config.yaml`
2. Start the service with `sudo systemctl start lmon`

#### Manual Installation

If you prefer to install the service manually:

1. Copy the binary to `/opt/lmon/lmon`
2. Copy the systemd service file to `/etc/systemd/system/lmon.service`
3. Create a user and group for the service: `sudo useradd -r -s /bin/false lmon`
4. Create the configuration directory: `sudo mkdir -p /etc/lmon`
5. Copy your configuration file to `/etc/lmon/config.yaml`
6. Enable and start the service:

```bash
sudo systemctl enable lmon
sudo systemctl start lmon
```

### Using Docker

Basic usage:
```bash
docker run -p 8080:8080 -v /path/to/config:/etc/lmon lmon
```

To get accurate system-wide CPU and memory metrics (not just container metrics):
```bash
docker run -p 8080:8080 -v /path/to/config:/etc/lmon --pid=host --privileged -v /proc:/proc:ro lmon
```

The `--pid=host` and `--privileged` flags allow the container to access the host's process namespace, and mounting `/proc` gives access to the host's process information.

### Using Docker Compose

Create a `docker-compose.yml` file:

```yaml
version: '3'

services:
  lmon:
    image: lmon
    # Alternatively, build from source:
    # build: .
    ports:
      - "8080:8080"
    volumes:
      - ./config:/etc/lmon
      # Mount proc for system-wide metrics:
      - /proc:/proc:ro
      # Mount root filesystem and other partitions for disk monitoring:
      - /:/hostroot:ro
      - /home:/hosthome:ro
      # Add additional partitions as needed:
      # - /var:/hostvar:ro
    environment:
      - LMON_WEB_HOST=0.0.0.0
      - LMON_WEB_PORT=8080
      - GIN_MODE=release
      # Optional: Configure webhook
      # - LMON_WEBHOOK_ENABLED=true
      # - LMON_WEBHOOK_URL=https://hooks.slack.com/services/XXX/YYY/ZZZ
    # For system-wide metrics:
    pid: host
    privileged: true
    restart: unless-stopped
    # Health check configuration
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 5s
```

Run with:

```bash
docker-compose up -d
```

This configuration includes all necessary settings for accurate system monitoring and provides a persistent service that restarts automatically.

### Using Podman

Podman is a daemonless container engine that can be used as a drop-in replacement for Docker. You can run lmon with Podman using similar commands:

Basic usage:
```bash
podman build -t lmon .
podman run -p 8080:8080 -v /path/to/config:/etc/lmon lmon
```

To get accurate system-wide CPU and memory metrics:
```bash
podman run -p 8080:8080 -v /path/to/config:/etc/lmon --pid=host --privileged -v /proc:/proc:ro --healthcheck-cmd="wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1" --healthcheck-interval=30s --healthcheck-timeout=10s --healthcheck-retries=3 --healthcheck-start-period=5s lmon
```

For rootless Podman, you may need to add `--userns=keep-id` to preserve user permissions:
```bash
podman run --userns=keep-id -p 8080:8080 -v /path/to/config:/etc/lmon --pid=host --privileged -v /proc:/proc:ro --healthcheck-cmd="wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1" --healthcheck-interval=30s --healthcheck-timeout=10s --healthcheck-retries=3 --healthcheck-start-period=5s lmon
```

You can also use Podman Compose with the same docker-compose.yml file:
```bash
podman-compose up -d
```

Note: Depending on your system configuration, you might need to adjust SELinux settings or use additional flags for volume mounts when using Podman.

## Web UI

The web UI is available at `http://localhost:8080` (or whatever host/port you've configured).

### Dashboard

The dashboard shows the status of all monitored items with traffic light indicators:
- Green: OK
- Yellow: Warning
- Red: Critical
- Gray: Unknown

Items in an unhealthy state are automatically expanded for visibility.

### Configuration

The configuration page allows you to:
- Add/remove disk monitoring
- Update CPU and memory thresholds
- Add/remove health check monitoring
- Configure webhook notifications

## Webhook Notifications

When a monitored item becomes unhealthy, lmon can send a notification to a webhook URL. The notification includes:
- Timestamp
- Item ID
- Item name
- Item type
- Status
- Value
- Message

This can be integrated with services like Slack, Discord, or custom notification systems.

## Testing

### Unit Tests

To run the unit tests:

```bash
go test ./...
```

### UI Tests

The project includes UI tests using the [go-rod](https://github.com/go-rod/rod) library to test the web interface. These tests verify:

- Dashboard page functionality
- Configuration page functionality
- Proper display of icons and formatting
- Specific UI fixes (root partition delete button, memory icon, percentage rounding)

To run the UI tests:

```bash
go test -v ./uitest
```

The tests will:
1. Start the lmon web server programmatically
2. Check that the server is healthy
3. Run the UI tests
4. Shut down the server when tests are complete

For more details, see the [UI testing documentation](./uitest/README.md).

## License

MIT
