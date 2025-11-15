// Package healthcheck provides the Healthcheck monitor implementation for HTTP endpoint checks.
// It supports both production and mock/test usage providers.
//
// # Healthcheck Monitor
//
// The Healthcheck monitor checks the health of an HTTP endpoint by making a request and evaluating the response.
//
// ## How it works:
//   - Uses a UsageProvider interface to abstract HTTP checks (default: Go's http.Client).
//   - Configured with:
//   - name: Logical name for the healthcheck.
//   - URL: URL to check.
//   - timeout: Timeout for the HTTP request.
//   - icon: UI icon (optional).
//   - On each check:
//   - Makes an HTTP GET request to the configured URL with the specified timeout.
//   - Status is:
//   - Green: HTTP 2xx response.
//   - Amber: HTTP 4xx response.
//   - Red: HTTP 5xx response or request error.
//   - Configuration is persisted back to the config struct for saving.
//
// Example usage:
//
//	hc, err := NewHealthcheck("My API", "https://api.example.com/health", 5000, "", nil)
//	result := hc.Check(context.Background())
package healthcheck

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"lmon/common"
	"lmon/config"
	"lmon/monitors"
	"lmon/monitors/docker"
)

const Icon = "activity" // Default icon for healthcheck monitors
const Group = "health"  // Group name for healthcheck monitors

// client is the default HTTP client used for health checks to support connection pooling and reuse.
var client = &http.Client{
	Timeout: 5 * time.Second, // Default timeout for HTTP requests
}

// UsageProvider is an interface for obtaining healthcheck usage statistics.
// It allows for production and mock implementations.
type UsageProvider interface {
	Check(ctx context.Context, path *url.URL, timeout int) (*http.Response, error)
}

// DockerProvider is an alias for docker.Provider to allow for dependency injection.
type DockerProvider = docker.Provider

// DefaultHealthcheckProvider is the default implementation of UsageProvider
// using Go's http.Client.
type DefaultHealthcheckProvider struct {
}

// NewDefaultDockerProvider creates a new Docker provider using the docker package
func NewDefaultDockerProvider() (DockerProvider, error) {
	return docker.NewDefaultDockerProvider()
}

// NewDefaultHealthcheckProvider creates a new DefaultHealthcheckProvider with the given timeout in milliseconds.
func NewDefaultHealthcheckProvider(msTimeout int) *DefaultHealthcheckProvider {
	if msTimeout == 0 {
		msTimeout = 5000 // Default to 5000ms (5 seconds) if no timeout is specified
	}
	return &DefaultHealthcheckProvider{}
}

// Check performs an HTTP GET request to the given URL with the specified timeout (ms).
// Returns the HTTP response or an error.
func (p *DefaultHealthcheckProvider) Check(ctx context.Context, path *url.URL, msTimeout int) (*http.Response, error) {
	to := time.Duration(msTimeout) * time.Millisecond
	toCtx, cancel := context.WithTimeout(ctx, to)
	defer cancel()
	req, err := http.NewRequestWithContext(toCtx, "GET", path.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error performing healthcheck: %s: %v", path.String(), err)
		return nil, err
	}
	if resp == nil {
		log.Printf("Healthcheck returned nil response for %s", path.String())
		return nil, fmt.Errorf("nil response from healthcheck for %s", path.String())
	}

	// read the body to ensure the request is complete
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	_, _ = io.Copy(io.Discard, resp.Body)

	return resp, err
}

// Healthcheck represents an HTTP endpoint health monitor.
type Healthcheck struct {
	name              string         // Logical name for the healthcheck
	timeout           int            // Timeout in milliseconds for the check
	respCode          int            // Expected response code to consider the check successful
	url               *url.URL       // URL to check
	icon              string         // Icon for UI display
	restartContainers string         // Optional: comma-separated list of containers to restart
	impl              UsageProvider  // Implementation for performing the check
	dockerImpl        DockerProvider // Implementation for Docker restart operations
}

// NewHealthcheck constructs a new Healthcheck monitor with the given parameters.
// If respCode is <= 0 it defaults to 200 (HTTP OK).
// If icon is empty, the default Icon is used.
// If impl is nil, the DefaultHealthcheckProvider is used.
// If dockerImpl is nil and restartContainers is set, a DefaultDockerProvider is created.
func NewHealthcheck(name string, urlRaw string, timeout int, respCode int, icon string, restartContainers string, impl UsageProvider, dockerImpl DockerProvider) (Healthcheck, error) {
	if icon == "" {
		icon = Icon
	}
	if respCode <= 0 {
		respCode = http.StatusOK // Default to HTTP 200 OK
	}
	if common.IsNil(impl) {
		impl = NewDefaultHealthcheckProvider(0)
	}
	if common.IsNil(dockerImpl) && restartContainers != "" {
		var err error
		dockerImpl, err = NewDefaultDockerProvider()
		if err != nil {
			return Healthcheck{}, fmt.Errorf("failed to create Docker provider: %w", err)
		}
	}
	parsedUrl, err := url.Parse(urlRaw)
	if err != nil {
		return Healthcheck{}, err
	}
	return Healthcheck{
		name:              name,
		url:               parsedUrl,
		icon:              icon,
		restartContainers: restartContainers,
		impl:              impl,
		dockerImpl:        dockerImpl,
		timeout:           timeout,
		respCode:          respCode,
	}, nil
}

// String returns a string representation of the Healthcheck monitor.
func (d Healthcheck) String() string {
	return fmt.Sprintf("Healthcheck{name: %s, URL: %s, timeout: %d, icon: %s}", d.name, d.url.String(), d.timeout, d.icon)
}

// DisplayName returns a human-readable name for the healthcheck monitor.
func (d Healthcheck) DisplayName() string {
	u := fmt.Sprintf("%s://%s", d.url.Scheme, d.url.Host)
	if d.respCode != http.StatusOK {
		return fmt.Sprintf("%s (%s - %d)", d.name, u, d.respCode)
	}
	return fmt.Sprintf("%s (%s)", d.name, u)
}

// Group returns the group/category for the healthcheck monitor.
func (d Healthcheck) Group() string {
	return Group
}

// Name returns the unique name/ID for the healthcheck monitor.
func (d Healthcheck) Name() string {
	return fmt.Sprintf("%s_%s", Group, d.name)
}

// Save persists the healthcheck monitor's configuration to the provided config struct.
func (d Healthcheck) Save(cfg *config.Config) {
	if d.respCode <= 0 {
		d.respCode = http.StatusOK // Default to HTTP 200 OK if not set
	}
	cfg.Monitoring.Healthcheck[d.name] = config.HealthcheckConfig{
		URL:               d.url.String(),
		Timeout:           d.timeout,
		RespCode:          d.respCode,
		Icon:              d.icon,
		RestartContainers: d.restartContainers,
	}
}

// HasRestartContainers returns true if this healthcheck has containers configured for restart
func (d Healthcheck) HasRestartContainers() bool {
	return d.restartContainers != ""
}

// RestartContainers restarts the containers associated with this healthcheck
func (d Healthcheck) RestartContainers(ctx context.Context) error {
	if !d.HasRestartContainers() {
		return fmt.Errorf("no containers configured for restart")
	}

	if d.dockerImpl == nil {
		return fmt.Errorf("docker provider not configured")
	}

	containerList := docker.ParseContainerList(d.restartContainers)
	return d.dockerImpl.RestartContainers(ctx, containerList)
}

// Check performs a healthcheck by making an HTTP request to the configured URL.
// Returns a Result with the status and value based on the HTTP response.
func (d Healthcheck) Check(ctx context.Context) monitors.Result {
	response, err := d.impl.Check(ctx, d.url, d.timeout*1000) // Convert seconds ms for the provider
	if err != nil {
		return monitors.Result{
			Status: monitors.RAGError,
			Value:  fmt.Sprintf("error getting healthcheck: %v", err),
		}
	}

	res := fmt.Sprintf("%d (%s)", response.StatusCode, response.Status)
	status := monitors.RAGGreen
	// status will be green if the response code matches the expected one or is a 2xx code.
	switch {
	case response.StatusCode == d.respCode:
		status = monitors.RAGGreen // Check if the response code matches the expected one and which is not 2xx
	case response.StatusCode < 300:
		status = monitors.RAGGreen // 2xx = always green
	case response.StatusCode >= 500:
		status = monitors.RAGRed // 5xx errors
	case response.StatusCode >= 300 && response.StatusCode < 500:
		status = monitors.RAGAmber // 4xx errors
	}

	return monitors.Result{
		Status: status,
		Value:  res,
	}
}


