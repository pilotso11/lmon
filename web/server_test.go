// server_test.go contains integration and unit tests for the lmon web server.
// These tests cover HTTP endpoints, configuration management, static file serving, and webhook integration.
package web

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"go.uber.org/goleak"

	"lmon/config"
	"lmon/monitors"
	"lmon/monitors/disk"
	"lmon/monitors/healthcheck"
	"lmon/monitors/mapper"
	"lmon/monitors/system"
)

// mockWebhookHandler is a test double for capturing webhook callback invocations.
type mockWebhookHandler struct {
	lastMessage atomic.String
	cnt         atomic.Int32
}

// webhookCallback stores the last message and increments the call count.
func (m *mockWebhookHandler) webhookCallback(msg string) {
	m.lastMessage.Store(msg)
	m.cnt.Inc()
}

// NewMockImplementations returns a set of mock monitor providers and a webhook callback for testing.
func NewMockImplementations(hook *mockWebhookHandler) *mapper.Implementations {
	return &mapper.Implementations{
		Disk:    disk.NewMockDiskProvider(50),
		Health:  healthcheck.NewMockHealthcheckProvider(50),
		Cpu:     system.NewMockCpuProvider(50),
		Mem:     system.NewMockMemProvider(50),
		Webhook: hook.webhookCallback,
	}
}

// startTestServer creates and starts a new test server instance with a temporary configuration.
// Returns the server and a mock webhook handler for assertions.
func startTestServer(ctx context.Context, t *testing.T, cfgFile string) (*Server, *mockWebhookHandler) {
	t.Helper()
	t.Setenv("LMON_WEB_PORT", "0")
	t.Setenv("LMON_WEB_HOST", "127.0.0.1")

	l := config.NewLoader(cfgFile, []string{t.TempDir()})
	cfg, err := l.Load()
	assert.NoError(t, err, "config loaded")
	push := monitors.NewMockPush()
	mon := monitors.NewService(ctx, 10*time.Millisecond, 10*time.Millisecond, push.Push)
	hook := &mockWebhookHandler{}
	s, err := NewServerWithContext(ctx, cfg, l, mon, mapper.NewMapper(NewMockImplementations(hook)))
	require.NoError(t, err, "server create")
	return s, hook
}

// sendRequest is a helper for sending HTTP requests to the test server.
// It marshals the data as JSON if provided, and returns the response and body as a string.
func sendRequest(ctx context.Context, t *testing.T, method string, s *Server, path string, timeout time.Duration, data any) (*http.Response, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var req *http.Request
	var err error
	if data != nil {
		bodybytes, err := json.Marshal(data)
		require.NoError(t, err, "marshal data")
		bodyBuff := bytes.NewBuffer(bodybytes)
		req, err = http.NewRequestWithContext(ctx, method, s.serverUrl+path, bodyBuff)
	} else {
		req, err = http.NewRequestWithContext(ctx, method, s.serverUrl+path, nil)
	}

	require.NoErrorf(t, err, "GET %s", s.serverUrl+path)

	client := http.Client{}
	res, err := client.Do(req)
	require.NoErrorf(t, err, "GET %s", s.serverUrl+path)

	return readBody(res, t)
}

// getRequest sends a GET request to the test server.
func getRequest(ctx context.Context, t *testing.T, s *Server, path string) (*http.Response, string) {
	return sendRequest(ctx, t, "GET", s, path, 20*time.Millisecond, nil)
}

// deleteRequest sends a DELETE request to the test server.
func deleteRequest(ctx context.Context, t *testing.T, s *Server, path string) (*http.Response, string) {
	return sendRequest(ctx, t, "DELETE", s, path, 10*time.Millisecond, nil)
}

// postRequest sends a POST request with JSON data to the test server.
func postRequest(ctx context.Context, t *testing.T, s *Server, path string, data any) (*http.Response, string) {
	return sendRequest(ctx, t, "POST", s, path, 100*time.Millisecond, data)
}

// readBody reads and closes the response body, returning the response and body as a string.
func readBody(res *http.Response, t *testing.T) (*http.Response, string) {
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(res.Body)
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalln(err)
	}
	assert.NoError(t, err, "read body")
	return res, string(body)
}

// TestNewServerWithContext_Smoke verifies that the server can be created, started, and stopped without panicking.
func TestNewServerWithContext_Smoke(t *testing.T) {
	defer goleak.VerifyNone(t)

	assert.NotPanics(t, func() {
		ctx, cancel := context.WithCancel(t.Context())
		s, _ := startTestServer(ctx, t, "")
		s.Start()

		time.Sleep(10 * time.Millisecond)

		cancel()
	})

	time.Sleep(10 * time.Millisecond)
}

// TestSelfHealthcheck checks that the /healthz endpoint returns HTTP 200 OK.
func TestSelfHealthcheck(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := startTestServer(ctx, t, "")
	s.Start()

	r, body := getRequest(ctx, t, s, "/healthz")

	assert.Equal(t, http.StatusOK, r.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)
}

// TestGetIndex checks that the root and /index.html endpoints return the dashboard HTML.
func TestGetIndex(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := startTestServer(ctx, t, "")
	s.Start()

	r, body := getRequest(ctx, t, s, "/")

	assert.Equal(t, http.StatusOK, r.StatusCode, "status code")
	assert.True(t, within(len(indexHtml), len(body), .10), "index returned is about the same length as the template")

	r, body = getRequest(ctx, t, s, "/index.html")

	assert.Equal(t, http.StatusOK, r.StatusCode, "status code")
	assert.True(t, within(len(indexHtml), len(body), .10), "index returned is about the same length as the template")
}

// within returns true if i2 is within the given tolerance of i.
func within(i int, i2 int, tolerance float64) bool {
	d := i - i2
	if d < 0 {
		d = -d
	}
	return float64(d)/float64(i) < tolerance
}

// TestGetConfig checks that the /config endpoint returns the configuration HTML.
func TestGetConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := startTestServer(ctx, t, "")
	s.Start()

	r, body := getRequest(ctx, t, s, "/config")

	assert.Equal(t, http.StatusOK, r.StatusCode, "status code")
	assert.True(t, within(len(configHtml), len(body), .10), "config returned is about the same length as the template")
}

//go:embed static/icon.svg
var icon string

// TestStatic checks that static files are served correctly.
func TestStatic(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := startTestServer(ctx, t, "")
	s.Start()

	r, body := getRequest(ctx, t, s, "/static/icon.svg")
	assert.Equal(t, http.StatusOK, r.StatusCode, "status code")
	assert.Equal(t, icon, body)
}

// TestSetSystemConfig verifies updating the system configuration via the API.
func TestSetSystemConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := startTestServer(ctx, t, "")
	s.Start()

	cfg := config.SystemConfig{
		CPU: config.SystemItem{
			Threshold: 55,
			Icon:      "cpu-icon",
		},
		Memory: config.SystemItem{
			Threshold: 66,
			Icon:      "mem-icon",
		},
		Title: "new title",
	}

	resp, body := postRequest(ctx, t, s, "/api/config/system", cfg)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	assert.Equal(t, cfg, s.config.Monitoring.System, "config applied")
}

// TestSetInterval verifies updating the monitoring interval via the API.
func TestSetInterval(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := startTestServer(ctx, t, "")
	s.Start()

	data := struct {
		Interval int
	}{
		Interval: 10,
	}

	resp, body := postRequest(ctx, t, s, "/api/config/interval", data)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	assert.Equal(t, 10, s.config.Monitoring.Interval, "config applied")
}

// TestAddDisk verifies adding a disk monitor via the API.
func TestAddDisk(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := startTestServer(ctx, t, "")
	s.Start()

	data := config.DiskConfig{
		Threshold: 77,
		Icon:      "disk-icon",
		Path:      ".",
	}
	id := "test-disk"
	resp, body := postRequest(ctx, t, s, "/api/config/disk/"+id, data)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	assert.Equal(t, 1, len(s.config.Monitoring.Disk), "number of disk entries")
	d2, ok := s.config.Monitoring.Disk[id]
	assert.True(t, ok, "disk entry exists")
	assert.Equal(t, data, d2, "disk entry applied")
}

// TestDeleteDisk verifies deleting a disk monitor via the API and handling of missing entries.
// TestDeleteDisk verifies deleting a disk monitor and handling repeated deletes (not found).
func TestDeleteDisk(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := startTestServer(ctx, t, "")
	s.Start()

	data := config.DiskConfig{
		Threshold: 77,
		Icon:      "disk-icon",
		Path:      ".",
	}
	id := "test-disk"
	resp, body := postRequest(ctx, t, s, "/api/config/disk/"+id, data)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	assert.Equal(t, 1, len(s.config.Monitoring.Disk), "number of disk entries")
	d2, ok := s.config.Monitoring.Disk[id]
	assert.True(t, ok, "disk entry exists")
	assert.Equal(t, data, d2, "disk entry applied")

	resp, body = deleteRequest(ctx, t, s, "/api/config/disk/"+id)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	assert.Equal(t, 0, len(s.config.Monitoring.Disk), "number of disk entries")

	// New: Ensure the disk is also removed from /api/items
	resp, body = getRequest(ctx, t, s, "/api/items")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	var stats map[string]monitors.Result
	err := json.Unmarshal([]byte(body), &stats)
	assert.NoError(t, err, "unmarshal")
	_, exists := stats["disk_"+id]
	assert.False(t, exists, "deleted disk should not be in /api/items")

	resp, body = deleteRequest(ctx, t, s, "/api/config/disk/"+id)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "status code")
}

// TestAddHealthcheck verifies adding a healthcheck monitor via the API.
func TestAddHealthcheck(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := startTestServer(ctx, t, "")
	s.Start()

	data := config.HealthcheckConfig{
		Timeout: 77,
		Icon:    "disk-icon",
		URL:     s.serverUrl + "/healthz",
	}
	id := "test-health"
	resp, body := postRequest(ctx, t, s, "/api/config/healthcheck/"+id, data)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	assert.Equal(t, 1, len(s.config.Monitoring.Healthcheck), "number of healthcheck entries")
	d2, ok := s.config.Monitoring.Healthcheck[id]
	assert.True(t, ok, "healthcheck entry exists")
	assert.Equal(t, data, d2, "healthcheck entry applied")
}

// TestDeleteHealthcheck verifies deleting a healthcheck monitor and handling repeated deletes (not found).
func TestDeleteHealthcheck(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := startTestServer(ctx, t, "")
	s.Start()

	data := config.HealthcheckConfig{
		Timeout: 77,
		Icon:    "disk-icon",
		URL:     s.serverUrl + "/healthz",
	}
	id := "test-health"
	resp, body := postRequest(ctx, t, s, "/api/config/healthcheck/"+id, data)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	assert.Equal(t, 1, len(s.config.Monitoring.Healthcheck), "number of healthcheck entries")
	d2, ok := s.config.Monitoring.Healthcheck[id]
	assert.True(t, ok, "healthcheck entry exists")
	assert.Equal(t, data, d2, "healthcheck entry applied")

	resp, body = deleteRequest(ctx, t, s, "/api/config/healthcheck/"+id)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	assert.Equal(t, 0, len(s.config.Monitoring.Healthcheck), "number of healthcheck entries")

	// New: Ensure the healthcheck is also removed from /api/items
	resp, body = getRequest(ctx, t, s, "/api/items")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	var stats map[string]monitors.Result
	err := json.Unmarshal([]byte(body), &stats)
	assert.NoError(t, err, "unmarshal")
	_, exists := stats["healthcheck_"+id]
	assert.False(t, exists, "deleted healthcheck should not be in /api/items")

	resp, body = deleteRequest(ctx, t, s, "/api/config/healthcheck/"+id)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "status code")

}

// TestSetWebhook verifies updating the webhook configuration via the API.
func TestSetWebhook(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := startTestServer(ctx, t, "")
	s.Start()

	data := config.WebhookConfig{
		Enabled: true,
		URL:     s.serverUrl + "/testhook",
	}

	resp, body := postRequest(ctx, t, s, "/api/config/webhook", data)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	assert.Equal(t, data, s.config.Webhook, "healthcheck entry applied")
}

// TestWebHookAndCallback verifies that webhook callbacks are triggered and received.
func TestWebHookAndCallback(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, hook := startTestServer(ctx, t, "")
	s.Start()

	data := config.WebhookConfig{
		Enabled: true,
		URL:     s.serverUrl + "/testhook",
	}

	resp, body := postRequest(ctx, t, s, "/api/config/webhook", data)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	assert.Equal(t, data, s.config.Webhook, "healthcheck entry applied")

	// Simulate a disk status change to trigger the webhook.
	s.mapper.Impls.Disk.Current.Store(99)
	d := config.DiskConfig{Threshold: 1, Icon: "", Path: "."}
	id := "test-disk"
	resp, body = postRequest(ctx, t, s, "/api/config/disk/"+id, d)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")

	time.Sleep(10 * time.Millisecond) // time for webhook to send

	assert.Equal(t, int32(1), hook.cnt.Load(), "got hook")
	assert.Contains(t, hook.lastMessage.Load(), "Red", "got hook")
}

//go:embed test/postsave.yaml
var expectedFile string

// TestConfigSaved verifies that configuration changes are persisted to disk.
func TestConfigSaved(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	dir := t.TempDir()
	testFile := strings.Join([]string{dir, "test.yaml"}, string(os.PathSeparator))

	s, _ := startTestServer(ctx, t, testFile)
	s.Start()

	data := struct {
		Interval int
	}{
		Interval: 10,
	}

	resp, body := postRequest(ctx, t, s, "/api/config/interval", data)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	assert.Equal(t, 10, s.config.Monitoring.Interval, "config applied")

	bodyBytes, err := os.ReadFile(testFile)
	assert.NoError(t, err, "read config file")

	assert.Equal(t, expectedFile, string(bodyBytes), "config saved")
}

// TestMapper_SetConfig verifies that SetConfig applies all monitor types and persists configuration.
func TestMapper_SetConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := startTestServer(ctx, t, "")

	l := config.NewLoader("config.yaml", []string{t.TempDir()})
	cfg, _ := l.Load()

	cfg.Monitoring.System.CPU.Threshold = 99
	cfg.Monitoring.System.Memory.Threshold = 99

	cfg.Monitoring.Disk["disk_test_d"] = config.DiskConfig{
		Threshold: 99,
		Icon:      "storage",
		Path:      "/",
	}
	cfg.Monitoring.Healthcheck["health_test_h"] = config.HealthcheckConfig{
		URL:     "http://localhost:8080/healtz",
		Timeout: 1,
		Icon:    "activity",
	}

	err := s.SetConfig(t.Context(), cfg.Monitoring)
	assert.NoError(t, err, "SetConfig()")
	assert.Equal(t, 4, s.monitor.Size(), "monitors setup")

	cfg.Monitoring.Interval = 10
	s.monitor.SetPeriod(ctx, 10*time.Second, 1*time.Second)

	cfg2, _ := l.Load()
	err = s.monitor.Save(cfg2)
	assert.NoError(t, err, "save")
	assert.Equal(t, cfg.Monitoring, cfg2.Monitoring, "config applied")
}

// TestGetItems verifies the /api/items endpoint and fetching individual items.
func TestGetItems(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	dir := t.TempDir()
	testFile := strings.Join([]string{dir, "test.yaml"}, string(os.PathSeparator))

	data, err := os.ReadFile("test/full.yaml")
	require.NoError(t, err, "read test config")
	err = os.WriteFile(testFile, data, 0644)
	assert.NoError(t, err, "write test config")

	s, _ := startTestServer(ctx, t, testFile)
	s.Start()

	resp, body := getRequest(ctx, t, s, "/api/items")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")

	var stats map[string]monitors.Result
	err = json.Unmarshal([]byte(body), &stats)
	assert.NoError(t, err, "unmarshal")

	assert.Equal(t, 4, len(stats), "stats entries")

	t.Run("getItem cpu", func(t *testing.T) {
		resp, body = getRequest(ctx, t, s, "/api/items/system/cpu")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	})
	t.Run("getItem mem", func(t *testing.T) {
		resp, body = getRequest(ctx, t, s, "/api/items/system/mem")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")

	})
	t.Run("getItem disk", func(t *testing.T) {
		resp, body = getRequest(ctx, t, s, "/api/items/disk/root")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")

	})
	t.Run("getItem healthcheck", func(t *testing.T) {
		resp, body = getRequest(ctx, t, s, "/api/items/healthcheck/google")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	})
	t.Run("getItem missing", func(t *testing.T) {
		resp, body = getRequest(ctx, t, s, "/api/items/disk/missing")
		assert.Equal(t, http.StatusNotFound, resp.StatusCode, "status code")
	})
}

// TestGetApiConfig verifies the /api/config endpoint returns the correct configuration.
func TestGetApiConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	dir := t.TempDir()
	testFile := strings.Join([]string{dir, "test.yaml"}, string(os.PathSeparator))

	data, err := os.ReadFile("test/full.yaml")
	require.NoError(t, err, "read test config")
	err = os.WriteFile(testFile, data, 0644)
	assert.NoError(t, err, "write test config")

	s, _ := startTestServer(ctx, t, testFile)
	s.Start()

	resp, body := getRequest(ctx, t, s, "/api/config")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")

	var cfg config.Config
	err = json.Unmarshal([]byte(body), &cfg)
	assert.NoError(t, err, "unmarshal")

	assert.Equal(t, 90, cfg.Monitoring.System.CPU.Threshold)
	assert.Equal(t, 90, cfg.Monitoring.System.Memory.Threshold)
	assert.Equal(t, 85, cfg.Monitoring.Disk["root"].Threshold)
	assert.Equal(t, "https://google.com", cfg.Monitoring.Healthcheck["google"].URL)
}

// TestBadJson verifies that invalid JSON in POST requests returns HTTP 400 Bad Request.
func TestBadJson(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := startTestServer(ctx, t, "")
	s.Start()

	resp, _ := postRequest(ctx, t, s, "/api/config/system", "not json")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "status code")

	resp, _ = postRequest(ctx, t, s, "/api/config/interval", "not json")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "status code")
}

// TestDeleteBadType verifies that deleting a monitor with an invalid type returns HTTP 400 Bad Request.
func TestDeleteBadType(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := startTestServer(ctx, t, "")
	s.Start()

	resp, _ := deleteRequest(ctx, t, s, "/api/config/bad/123")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "status code")

}
