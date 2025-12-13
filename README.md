# lmon - Lightweight Monitoring Service
![go](https://github.com/pilotso11/lmon/actions/workflows/go.yml/badge.svg)
![docker](https://github.com/pilotso11/lmon/actions/workflows/docker.yml/badge.svg)
![Coverage](https://img.shields.io/badge/Coverage-88.2%25-brightgreen)

A lightweight, extensible monitoring service written in Go. lmon monitors system resources, disk usage, and application health, providing a modern web UI and flexible configuration.


---

## Features

- Monitor disk usage for any filesystem path
- Monitor CPU and memory usage with configurable thresholds
- Monitor HTTP endpoints with health checks
- Monitor network connectivity via ICMP ping
- Monitor Docker container restart counts with restart capability
- Web UI dashboard with traffic light status indicators
- Add/remove monitors and update thresholds via the web UI
- Webhook notifications for unhealthy states
- Configuration via YAML file and/or environment variables
- Systemd service support (auto/manual install)
- Docker and Podman support (including Compose)

---

## Requirements

- Go 1.24 or later (for build)
- Linux, macOS, or Windows for system monitoring

---

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

---

## Configuration

lmon uses a YAML configuration file and/or environment variables. By default, it loads `config.yaml` from the current directory, 
or `/etc/lmon/config.yaml` if present. You can override the config path using the 
`LMON_CONFIG_FILE` or `LMON_CONFIG_PATH` environment variables.  If `LMON_CONFIG_FILE` includes a path it will 
override any other specified path.

### Example Configuration (`config.yaml`)

```yaml
web:
  host: 0.0.0.0
  port: 8080

monitoring:
  interval: 60
  disk:
    root:
      path: /
      threshold: 80
      icon: hdd
    home:
      path: /home
      threshold: 80
      icon: hdd-network
  system:
    cpu:
      threshold: 90
      icon: cpu
    memory:
      threshold: 90
      icon: speedometer
    title: "lmon Dashboard"
  healthcheck:
    self:
      url: http://localhost:8080/healthz
      timeout: 10
      icon: activity
    google:
      url: https://google.com
      timeout: 10
      icon: heart-pulse
  ping:
    gateway:
      address: 192.168.1.1
      timeout: 1000
      amberThreshold: 100
      icon: wifi
  docker:
    app_containers:
      containers: "web-app, api-server, worker"
      threshold: 5
      icon: box

webhook:
  enabled: true
  url: http://localhost:8080/testhook
```

**Notes:**
- Disk and healthcheck monitors are keyed by name (e.g., `root`, `home`, `self`, `google`).
- `system.cpu` and `system.memory` thresholds are percentages.
- `ping.amberThreshold` is the response time in milliseconds that triggers amber (warning) status.
- `docker.containers` can be a comma or space-separated list of container names or IDs.
- `docker.threshold` is the maximum restart count before alerting.
- `webhook.enabled` and `webhook.url` control notification integration.

### Docker Monitoring

The Docker monitor tracks container restart counts and can restart containers via the web UI:

- **Requirements**: Docker socket access (typically `/var/run/docker.sock`)
- **Containers**: Specify container names or IDs (comma or space-separated)
- **Threshold**: Maximum restart count before triggering alerts
  - Green: Below 90% of threshold
  - Amber: Between 90% and threshold
  - Red: At or above threshold
- **Restart Action**: Click the restart button in the web UI to restart all containers in the monitor

**Security Note**: Docker socket access grants significant privileges. Ensure lmon runs with appropriate permissions and consider using read-only socket access where possible.

### Environment Variables

All config options can be set with the `LMON_` prefix. Examples:
- `LMON_WEB_HOST=127.0.0.1`
- `LMON_WEB_PORT=8080`
- `LMON_MONITORING_INTERVAL=30`
- `LMON_WEBHOOK_ENABLED=true`
- `LMON_WEBHOOK_URL=https://hooks.slack.com/services/...`
- `LMON_MONITORING_DISK_NAS_PATH=/mnt/nas`
- `LMON_MONITORING_DISK_NAS_THRESHOLD=90`
- `LMON_MONITORING_DISK_NAS_ICON=hdd-network`

---

## Running

### Command Line

```bash
./lmon
```

### As a Systemd Service

#### Manual Installation

1. Copy the binary to `/opt/lmon/lmon`
2. Copy `lmon.service` to `/etc/systemd/system/lmon.service`
3. Create the user/group: `sudo useradd -r -s /bin/false lmon`
4. Create `/etc/lmon` and copy your config
5. Enable and start:

```bash
sudo systemctl enable lmon
sudo systemctl start lmon
```

---

### Using Docker

Basic usage:
```bash
docker run -p 8080:8080 -v /path/to/config:/etc/lmon lmon
```

For accurate host metrics:
```bash
docker run -p 8080:8080 -v /path/to/config:/etc/lmon --pid=host --privileged -v /proc:/proc:ro lmon
```

### Using Docker Compose

Example `docker-compose.yml`:

```yaml
version: '3'
services:
  lmon:
    image: lmon
    ports:
      - "8080:8080"
    volumes:
      - ./config:/etc/lmon
      - /proc:/proc:ro
      - /:/host/root:ro
      - /home:/host/home:ro
      - /var/run/docker.sock:/var/run/docker.sock  # For Docker container restart
    environment:
      - LMON_WEB_HOST=0.0.0.0
      - LMON_WEB_PORT=8080
    pid: host
    privileged: true
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 5s

```

**Note:** To enable Docker container restart functionality, mount the Docker socket as shown above. The container needs appropriate permissions to access it:

```yaml
# Option 1: Run as root (simplest, but less secure)
user: "0:0"

# Option 2: Add user to docker group (recommended)
# First, get your docker group ID: getent group docker | cut -d: -f3
# Then add to compose file:
user: "1000:999"  # Replace 999 with your docker group ID
```

Run with:

```bash
docker-compose up -d
```

### Using Podman

Podman is supported as a drop-in replacement for Docker. Use similar flags as above.

---

## Web UI

- Access at `http://localhost:8080` (or your configured host/port).
- **Dashboard:** Shows all monitored items with traffic light status (green/yellow/red/gray).
- **Configuration:** Add/remove disk and healthcheck monitors, update thresholds, and configure webhook notifications.
- **Mobile View:** Optimized for small screens.

![Dashboard](screenshots/dashboard.png)
![Configuration](screenshots/config.png)

---

## Webhook Notifications

When a monitored item becomes unhealthy, lmon can send a JSON notification to the configured webhook URL. The payload includes:
- Timestamp
- Item ID and name
- Item type
- Status and value
- Message

Integrate with Slack, Discord, or custom endpoints.

---

## Testing

### Unit Tests

```bash
go test -race ./...
```

### UI Tests

UI tests use [go-rod](https://github.com/go-rod/rod) to verify dashboard and config functionality - they are run as part of the full test suite. 
To run independently, use:

```bash
go test -v ./uitest
```


---

## License

MIT

---
