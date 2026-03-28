# Use multi-stage build with cross-compilation
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

# Install CA certificates for downloads
RUN apk add --no-cache ca-certificates && update-ca-certificates

WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Cross-compile for target platform
ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-w -s" -o lmon

# Use a minimal alpine image for the final image
FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/lmon /app/lmon

# Create directories for configuration and static files
RUN mkdir -p /etc/lmon

# Expose the web server port
EXPOSE 8080

# Set environment variables
ENV LMON_WEB_HOST=0.0.0.0
ENV LMON_WEB_PORT=8080
ENV LMON_CONFIG_FILE=/etc/lmon/config.yaml

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
