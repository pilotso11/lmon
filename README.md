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
    cpu_icon: "memory"
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

```bash
docker run -p 8080:8080 -v /path/to/config:/etc/lmon lmon
```

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

## License

MIT
