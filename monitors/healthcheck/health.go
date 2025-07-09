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
//   - url: URL to check.
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

// DefaultHealthcheckProvider is the default implementation of UsageProvider
// using Go's http.Client.
type DefaultHealthcheckProvider struct {
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
	name    string        // Logical name for the healthcheck
	timeout int           // Timeout in milliseconds for the check
	url     *url.URL      // URL to check
	icon    string        // Icon for UI display
	impl    UsageProvider // Implementation for performing the check
}

// NewHealthcheck constructs a new Healthcheck monitor with the given parameters.
// If icon is empty, the default Icon is used.
// If impl is nil, the DefaultHealthcheckProvider is used.
func NewHealthcheck(name string, urlRaw string, timeout int, icon string, impl UsageProvider) (Healthcheck, error) {
	if icon == "" {
		icon = Icon
	}
	if common.IsNil(impl) {
		impl = NewDefaultHealthcheckProvider(0)
	}
	parsedUrl, err := url.Parse(urlRaw)
	if err != nil {
		return Healthcheck{}, err
	}
	return Healthcheck{
		name:    name,
		url:     parsedUrl,
		icon:    icon,
		impl:    impl,
		timeout: timeout,
	}, nil
}

// String returns a string representation of the Healthcheck monitor.
func (d Healthcheck) String() string {
	return fmt.Sprintf("Healthcheck{name: %s, url: %s, timeout: %d, icon: %s}", d.name, d.url.String(), d.timeout, d.icon)
}

// DisplayName returns a human-readable name for the healthcheck monitor.
func (d Healthcheck) DisplayName() string {
	u := fmt.Sprintf("%s://%s", d.url.Scheme, d.url.Host)
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
	cfg.Monitoring.Healthcheck[d.name] = config.HealthcheckConfig{
		URL:     d.url.String(),
		Timeout: d.timeout,
		Icon:    d.icon,
	}
}

// Check performs a healthcheck by making an HTTP request to the configured URL.
// Returns a Result with the status and value based on the HTTP response.
func (d Healthcheck) Check(ctx context.Context) monitors.Result {
	response, err := d.impl.Check(ctx, d.url, d.timeout*1000) // Convert ms to seconds for the provider
	if err != nil {
		return monitors.Result{
			Status: monitors.RAGError,
			Value:  fmt.Sprintf("error getting healthcheck: %v", err),
		}
	}

	res := fmt.Sprintf("%d (%s)", response.StatusCode, response.Status)
	status := monitors.RAGGreen
	switch {
	case response.StatusCode >= 500:
		status = monitors.RAGRed
	case response.StatusCode >= 300 && response.StatusCode < 500:
		status = monitors.RAGAmber
	}

	return monitors.Result{
		Status: status,
		Value:  res,
	}
}
