FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o lmon

# Use a minimal alpine image for the final image
FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/lmon /app/lmon

# Create directories for configuration and static files
RUN mkdir -p /etc/lmon/config

# Expose the web server port
EXPOSE 8080

# Set environment variables
ENV LMON_WEB_HOST=0.0.0.0
ENV LMON_WEB_PORT=8080
ENV LMON_CONFIG_PATH=/app/lmon/config

# Note: To get accurate system-wide CPU and memory metrics, run the container with:
# docker run --pid=host --privileged -v /proc:/proc:ro -v /:/hostroot:ro ...
# 
# For monitoring disk partitions, mount the parent drives you want to monitor:
# docker run -v /:/hostroot:ro -v /home:/hosthome:ro ...

# Add health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

# Run the application
CMD ["/app/lmon"]
