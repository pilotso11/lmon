// server_test.go contains integration and unit tests for the lmon web server.
// These tests cover HTTP endpoints, configuration management, static file serving, and webhook integration.
package web

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	"lmon/config"
	"lmon/monitors"
)

// TestNewServerWithContext_Smoke verifies that the server can be created, started, and stopped without panicking.
func TestNewServerWithContext_Smoke(t *testing.T) {
	defer goleak.VerifyNone(t)

	assert.NotPanics(t, func() {
		ctx, cancel := context.WithCancel(t.Context())
		s, _ := StartTestServer(ctx, t, "")
		s.Start(ctx)

		time.Sleep(10 * time.Millisecond)

		cancel()
	})

	time.Sleep(50 * time.Millisecond)
}

// TestSelfHealthcheck checks that the /healthz endpoint returns HTTP 200 OK.
func TestSelfHealthcheck(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	r, body := GetTestRequest(ctx, t, s, "/healthz")

	assert.Equal(t, http.StatusOK, r.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)
}

// TestGetIndex checks that the root and /index.html endpoints return the dashboard HTML.
func TestGetIndex(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	r, body := GetTestRequest(ctx, t, s, "/")

	assert.Equal(t, http.StatusOK, r.StatusCode, "status code")
	assert.Less(t, 7000, len(body), "index returned is about the same length as the template")

	r, body = GetTestRequest(ctx, t, s, "/index.html")

	assert.Equal(t, http.StatusOK, r.StatusCode, "status code")
	assert.Less(t, 7000, len(body), "index returned is about the same length as the template")
}

// TestGetConfig checks that the /config endpoint returns the configuration HTML.
func TestGetConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	r, body := GetTestRequest(ctx, t, s, "/config")

	assert.Equal(t, http.StatusOK, r.StatusCode, "status code")
	assert.Less(t, 7000, len(body), "config returned is about the same length as the template")
}

//go:embed static/icons/icon.svg
var icon string

// TestStatic checks that static files are served correctly.
func TestStatic(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	r, body := GetTestRequest(ctx, t, s, "/static/icons/icon.svg")
	assert.Equal(t, http.StatusOK, r.StatusCode, "status code")
	assert.Equal(t, icon, body)
}

// TestSetSystemConfig verifies updating the system configuration via the API.
func TestSetSystemConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

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

	resp, body := PostTestRequest(ctx, t, s, "/api/config/system", cfg)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	assert.Equal(t, cfg, s.config.Monitoring.System, "config applied")
}

// TestSetInterval verifies updating the monitoring interval via the API.
func TestSetInterval(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	data := struct {
		Interval int
	}{
		Interval: 10,
	}

	resp, body := PostTestRequest(ctx, t, s, "/api/config/interval", data)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	assert.Equal(t, 10, s.config.Monitoring.Interval, "config applied")
}

// TestAddDisk verifies adding a disk monitor via the API.
func TestAddDisk(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	data := config.DiskConfig{
		Threshold: 77,
		Icon:      "disk-icon",
		Path:      ".",
	}
	id := "test-disk"
	resp, body := PostTestRequest(ctx, t, s, "/api/config/disk/"+id, data)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	assert.Equal(t, 1, len(s.config.Monitoring.Disk), "number of disk entries")
	d2, ok := s.config.Monitoring.Disk[id]
	assert.True(t, ok, "disk entry exists")
	assert.Equal(t, data, d2, "disk entry applied")
}

// TestDeleteDisk verifies deleting a disk monitor via the API and handling of missing entries.
func TestDeleteDisk(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	data := config.DiskConfig{
		Threshold: 77,
		Icon:      "disk-icon",
		Path:      ".",
	}
	id := "test-disk"
	resp, body := PostTestRequest(ctx, t, s, "/api/config/disk/"+id, data)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	assert.Equal(t, 1, len(s.config.Monitoring.Disk), "number of disk entries")
	d2, ok := s.config.Monitoring.Disk[id]
	assert.True(t, ok, "disk entry exists")
	assert.Equal(t, data, d2, "disk entry applied")

	resp, body = DeleteTestRequest(ctx, t, s, "/api/config/disk/"+id)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	assert.Equal(t, 0, len(s.config.Monitoring.Disk), "number of disk entries")

	// New: Ensure the disk is also removed from /api/items
	resp, body = GetTestRequest(ctx, t, s, "/api/items")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	var stats map[string]monitors.Result
	err := json.Unmarshal([]byte(body), &stats)
	assert.NoError(t, err, "unmarshal")
	_, exists := stats["disk_"+id]
	assert.False(t, exists, "deleted disk should not be in /api/items")

	resp, body = DeleteTestRequest(ctx, t, s, "/api/config/disk/"+id)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "status code")
}

// TestAddHealthcheck verifies adding a healthcheck monitor via the API.
func TestAddHealthcheck(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	data := config.HealthcheckConfig{
		Timeout: 77,
		Icon:    "disk-icon",
		URL:     s.ServerUrl + "/healthz",
	}
	id := "test-health"
	resp, body := PostTestRequest(ctx, t, s, "/api/config/health/"+id, data)
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
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	data := config.HealthcheckConfig{
		Timeout: 77,
		Icon:    "disk-icon",
		URL:     s.ServerUrl + "/healthz",
	}
	id := "test-health"
	resp, body := PostTestRequest(ctx, t, s, "/api/config/health/"+id, data)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	assert.Equal(t, 1, len(s.config.Monitoring.Healthcheck), "number of healthcheck entries")
	d2, ok := s.config.Monitoring.Healthcheck[id]
	assert.True(t, ok, "healthcheck entry exists")
	assert.Equal(t, data, d2, "healthcheck entry applied")

	resp, body = DeleteTestRequest(ctx, t, s, "/api/config/health/"+id)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	assert.Equal(t, 0, len(s.config.Monitoring.Healthcheck), "number of healthcheck entries")

	// New: Ensure the healthcheck is also removed from /api/items
	resp, body = GetTestRequest(ctx, t, s, "/api/items")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	var stats map[string]monitors.Result
	var err error
	var exists bool
	stats = map[string]monitors.Result{}
	err = json.Unmarshal([]byte(body), &stats)
	assert.NoError(t, err, "unmarshal")
	_, exists = stats["health_"+id]
	assert.False(t, exists, "deleted healthcheck should not be in /api/items")

	resp, body = DeleteTestRequest(ctx, t, s, "/api/config/health/"+id)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "status code")
}

// TestSetWebhook verifies updating the webhook configuration via the API.
func TestSetWebhook(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	data := config.WebhookConfig{
		Enabled: true,
		URL:     s.ServerUrl + "/testhook",
	}

	resp, body := PostTestRequest(ctx, t, s, "/api/config/webhook", data)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	assert.Equal(t, data, s.config.Webhook, "healthcheck entry applied")
}

// TestWebHookAndCallback verifies that webhook callbacks are triggered and received.
func TestWebHookAndCallback(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, hook := StartTestServer(ctx, t, "")
	s.Start(ctx)

	data := config.WebhookConfig{
		Enabled: true,
		URL:     s.ServerUrl + "/testhook",
	}

	resp, body := PostTestRequest(ctx, t, s, "/api/config/webhook", data)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	assert.Equal(t, data, s.config.Webhook, "healthcheck entry applied")

	// Simulate a disk status change to trigger the webhook.
	s.mapper.Impls.Disk.Current.Store(99)
	d := config.DiskConfig{Threshold: 1, Icon: "", Path: "."}
	id := "test-disk"
	resp, body = PostTestRequest(ctx, t, s, "/api/config/disk/"+id, d)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")

	time.Sleep(10 * time.Millisecond) // time for webhook to send

	assert.Equal(t, int32(1), hook.Cnt.Load(), "got hook")
	assert.Contains(t, hook.LastMessage.Load(), "Red", "got hook")
}

// TestPingMonitorAPI verifies adding, fetching, and deleting ping monitors via the API.
func TestPingMonitorAPI(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	// Add a ping monitor
	data := config.PingConfig{
		Address:        "127.0.0.1",
		Timeout:        1000,
		Icon:           "wifi",
		AmberThreshold: 50,
	}
	id := "test-ping"
	resp, body := PostTestRequest(ctx, t, s, "/api/config/ping/"+id, data)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	// Check config
	pc, ok := s.config.Monitoring.Ping[id]
	assert.True(t, ok, "ping entry exists")
	assert.Equal(t, data, pc, "ping entry applied")

	// Check /api/items
	var stats map[string]monitors.Result
	assert.Eventually(t, func() bool {
		resp, body = GetTestRequest(ctx, t, s, "/api/items")
		if resp.StatusCode != http.StatusOK {
			return false
		}
		stats = map[string]monitors.Result{}
		err := json.Unmarshal([]byte(body), &stats)
		if err != nil {
			return false
		}
		_, exists := stats["ping_"+id]
		return exists
	}, time.Second, 10*time.Millisecond, "Ping monitor should exist in /api/items")

	// Delete the ping monitor
	// Remove uses the monitor name, which is just id, but /api/items uses "ping_"+id
	resp, body = DeleteTestRequest(ctx, t, s, "/api/config/ping/"+id)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)
	assert.Equal(t, 0, len(s.config.Monitoring.Ping), "number of ping entries")

	// Ensure it's removed from /api/items
	resp, body = GetTestRequest(ctx, t, s, "/api/items")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	stats = map[string]monitors.Result{}
	err := json.Unmarshal([]byte(body), &stats)
	assert.NoError(t, err, "unmarshal")
	_, exists := stats["ping_"+id]
	assert.False(t, exists, "deleted ping should not be in /api/items")

	// Try to add a ping monitor with missing amberThreshold
	badData := config.PingConfig{
		Address: "127.0.0.1",
		Timeout: 1000,
		Icon:    "wifi",
		// amberThreshold missing
	}
	id = "badping"
	resp, body = PostTestRequest(ctx, t, s, "/api/config/ping/"+id, badData)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")

	// Try to delete a non-existent ping monitor
	resp, body = DeleteTestRequest(ctx, t, s, "/api/config/ping/nonexistent")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "status code")
}

// TestPingMonitorStatusTransitionsAndWebhook mocks status transitions for Ping monitor and checks webhook notifications.
func TestPingMonitorStatusTransitionsAndWebhook(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, hook := StartTestServer(ctx, t, "")
	s.monitor.SetPeriod(ctx, 10*time.Millisecond, 10*time.Millisecond) // Set a short period for testing
	s.Start(ctx)

	// Enable webhook for test
	webhookCfg := config.WebhookConfig{
		Enabled: true,
		URL:     s.ServerUrl + "/testhook",
	}
	resp, _ := PostTestRequest(ctx, t, s, "/api/config/webhook", webhookCfg)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	id := "ping-e2e"
	amber := 100

	// Green status
	s.mapper.Impls.Ping.ResponseMs.Store(50)
	s.mapper.Impls.Ping.Err.Store(nil)

	data := config.PingConfig{
		Address:        "127.0.0.1",
		Timeout:        1000,
		Icon:           "wifi",
		AmberThreshold: amber,
	}
	resp, _ = PostTestRequest(ctx, t, s, "/api/config/ping/"+id, data)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	_, exists := s.config.Monitoring.Ping[id]
	assert.True(t, exists, "Ping monitor should exist in config after API add")

	// Reload config from disk and validate
	reloadedCfg, err := s.loader.Load()
	assert.NoError(t, err, "should reload config after API add")
	_, exists = reloadedCfg.Monitoring.Ping[id]
	assert.True(t, exists, "Ping monitor should exist in config after save and reload")

	time.Sleep(20 * time.Millisecond)
	var stats map[string]monitors.Result
	_, body := GetTestRequest(ctx, t, s, "/api/items")
	stats = map[string]monitors.Result{}
	_ = json.Unmarshal([]byte(body), &stats)
	assert.Equal(t, monitors.RAGGreen.String(), stats["ping_"+id].Status.String(), "should see green status")
	assert.Equal(t, "50 ms", stats["ping_"+id].Value, "should see green value")

	// No webhook notification is expected for initial Green status.

	// Amber status
	s.mapper.Impls.Ping.ResponseMs.Store(150)
	s.mapper.Impls.Ping.Err.Store(nil)
	time.Sleep(20 * time.Millisecond)
	_, body = GetTestRequest(ctx, t, s, "/api/items")
	stats = map[string]monitors.Result{}
	_ = json.Unmarshal([]byte(body), &stats)
	assert.Equal(t, monitors.RAGAmber.String(), stats["ping_"+id].Status.String(), "should see amber status")
	assert.Equal(t, "150 ms", stats["ping_"+id].Value, "should see amber value")

	assert.Eventually(t, func() bool {
		return hook.LastMessage.Load() != "" &&
			strings.Contains(hook.LastMessage.Load(), "Amber")
	}, 50*time.Millisecond, 1*time.Millisecond, "webhook should contain Amber")

	// Red status
	s.mapper.Impls.Ping.Err.Store(&assert.AnError)
	time.Sleep(20 * time.Millisecond)
	_, body = GetTestRequest(ctx, t, s, "/api/items")
	stats = map[string]monitors.Result{}
	_ = json.Unmarshal([]byte(body), &stats)
	assert.Equal(t, monitors.RAGRed.String(), stats["ping_"+id].Status.String(), "should see red status")
	assert.Greater(t, len(stats["ping_"+id].Value), 0, "should see error value")

	assert.Eventually(t, func() bool {
		return hook.LastMessage.Load() != "" &&
			strings.Contains(hook.LastMessage.Load(), "Red")
	}, 50*time.Millisecond, 1*time.Millisecond, "webhook should contain Red")
}
