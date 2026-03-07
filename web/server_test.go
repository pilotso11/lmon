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
			Threshold:      55,
			Icon:           "cpu-icon",
			AlertThreshold: 1, // Default value
		},
		Memory: config.SystemItem{
			Threshold:      66,
			Icon:           "mem-icon",
			AlertThreshold: 1, // Default value
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
		Threshold:      77,
		Icon:           "disk-icon",
		Path:           ".",
		AlertThreshold: 1, // Default value
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
		Threshold:      77,
		Icon:           "disk-icon",
		Path:           ".",
		AlertThreshold: 1, // Default value
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
		Timeout:        77,
		Icon:           "disk-icon",
		URL:            s.ServerUrl + "/healthz",
		RespCode:       200,
		AlertThreshold: 1, // Default value
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
		Timeout:        77,
		Icon:           "disk-icon",
		URL:            s.ServerUrl + "/healthz",
		RespCode:       200,
		AlertThreshold: 1, // Default value
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
		AlertThreshold: 1, // Default value
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

	// Red status - transition from Amber to Red
	// With alert thresholds, this does NOT trigger a new webhook because both are failure states
	s.mapper.Impls.Ping.Err.Store(&assert.AnError)
	time.Sleep(20 * time.Millisecond)
	_, body = GetTestRequest(ctx, t, s, "/api/items")
	stats = map[string]monitors.Result{}
	_ = json.Unmarshal([]byte(body), &stats)
	assert.Equal(t, monitors.RAGRed.String(), stats["ping_"+id].Status.String(), "should see red status")
	assert.Greater(t, len(stats["ping_"+id].Value), 0, "should see error value")

	// No new webhook for Amber -> Red transition (both are failures)
	// The last webhook message should still be "Amber"
	assert.Contains(t, hook.LastMessage.Load(), "Amber", "webhook should still contain last Amber message")
	
	// Recovery to Green should trigger a webhook
	s.mapper.Impls.Ping.ResponseMs.Store(50)
	s.mapper.Impls.Ping.Err.Store(nil)
	time.Sleep(20 * time.Millisecond)
	_, body = GetTestRequest(ctx, t, s, "/api/items")
	stats = map[string]monitors.Result{}
	_ = json.Unmarshal([]byte(body), &stats)
	assert.Equal(t, monitors.RAGGreen.String(), stats["ping_"+id].Status.String(), "should see green status")
	
	assert.Eventually(t, func() bool {
		return hook.LastMessage.Load() != "" &&
			strings.Contains(hook.LastMessage.Load(), "Green")
	}, 50*time.Millisecond, 1*time.Millisecond, "webhook should contain Green for recovery")
}

// TestHandleGetItem tests the handleGetItem endpoint for both success and error cases
func TestHandleGetItem(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	// Add a disk monitor to have an item to retrieve
	diskConfig := config.DiskConfig{Path: "/tmp", Threshold: 80}
	monitor, err := s.mapper.NewDisk(ctx, "test-disk", diskConfig)
	assert.NoError(t, err)
	s.monitor.Add(ctx, monitor)

	// Wait for initial check
	time.Sleep(10 * time.Millisecond)

	t.Run("success", func(t *testing.T) {
		r, body := GetTestRequest(ctx, t, s, "/api/items/disk/test-disk")
		assert.Equal(t, http.StatusOK, r.StatusCode)

		var result map[string]interface{}
		err := json.Unmarshal([]byte(body), &result)
		assert.NoError(t, err)
		assert.Contains(t, result, "ID")
		assert.Equal(t, "disk_test-disk", result["ID"])
	})

	t.Run("item_not_found", func(t *testing.T) {
		r, _ := GetTestRequest(ctx, t, s, "/api/items/disk/nonexistent")
		assert.Equal(t, http.StatusBadRequest, r.StatusCode)
	})

	t.Run("different_group_not_found", func(t *testing.T) {
		r, _ := GetTestRequest(ctx, t, s, "/api/items/nonexistent/test-disk")
		assert.Equal(t, http.StatusBadRequest, r.StatusCode)
	})
}

// TestHandleGetConfig tests the handleGetConfig endpoint
func TestHandleGetConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	r, body := GetTestRequest(ctx, t, s, "/api/config")
	assert.Equal(t, http.StatusOK, r.StatusCode)

	var config map[string]interface{}
	err := json.Unmarshal([]byte(body), &config)
	assert.NoError(t, err)
	assert.Contains(t, config, "Monitoring")
	assert.Contains(t, config, "Web")
}

// TestWriteJson tests the writeJson function with various data types and error conditions
func TestWriteJson(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)
	defer s.Stop()

	t.Run("success_simple_data", func(t *testing.T) {
		// Use a test endpoint that calls writeJson
		r, body := GetTestRequest(ctx, t, s, "/api/config")
		assert.Equal(t, http.StatusOK, r.StatusCode)
		assert.Contains(t, r.Header.Get("Content-Type"), "application/json")
		assert.NotEmpty(t, body)
	})

	t.Run("marshal_error_with_invalid_data", func(t *testing.T) {
		// Test by sending malformed JSON to trigger unmarshallBody error path
		req := `{"invalid": "malformed json"`
		r, _ := PostTestRequest(ctx, t, s, "/api/config/system", req)
		assert.Equal(t, http.StatusBadRequest, r.StatusCode) // Should return 400 for bad JSON
	})
}

// TestUnmarshallBody tests the unmarshallBody function with various request scenarios
func TestUnmarshallBody(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)
	defer s.Stop()

	t.Run("success", func(t *testing.T) {
		validJSON := `{"interval": 30}`
		r, _ := PostTestRequest(ctx, t, s, "/api/config/interval", validJSON)
		// Should succeed in unmarshalling (though may fail validation later)
		assert.NotEqual(t, http.StatusInternalServerError, r.StatusCode)
	})

	t.Run("invalid_json_format", func(t *testing.T) {
		invalidJSON := `{"interval": 30` // Missing closing brace
		r, _ := PostTestRequest(ctx, t, s, "/api/config/interval", invalidJSON)
		assert.Equal(t, http.StatusBadRequest, r.StatusCode)
	})

	t.Run("malformed_json_data", func(t *testing.T) {
		malformedJSON := `{"interval": "not_a_number"}`
		r, _ := PostTestRequest(ctx, t, s, "/api/config/interval", malformedJSON)
		// Should succeed in unmarshalling but may fail validation
		assert.NotEqual(t, http.StatusInternalServerError, r.StatusCode)
	})
}

// TestSetConfig tests the SetConfig function with various configurations
func TestSetConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")

	t.Run("success_complete_config", func(t *testing.T) {
		cfg := config.MonitoringConfig{
			Interval: 30,
			System: config.SystemConfig{
				CPU:    config.SystemItem{Threshold: 80},
				Memory: config.SystemItem{Threshold: 85},
			},
			Disk: map[string]config.DiskConfig{
				"root": {Path: "/", Threshold: 90},
				"tmp":  {Path: "/tmp", Threshold: 85},
			},
			Healthcheck: map[string]config.HealthcheckConfig{
				"google": {URL: "https://www.google.com", Timeout: 5000},
				"local":  {URL: "http://localhost:8080/healthz", Timeout: 1000},
			},
			Ping: map[string]config.PingConfig{
				"google-dns": {Address: "8.8.8.8", AmberThreshold: 100},
				"cloudflare": {Address: "1.1.1.1", AmberThreshold: 100},
			},
		}

		err := s.SetConfig(ctx, cfg)
		assert.NoError(t, err)

		// Verify basic monitors were added - wait for background checks to populate results
		assert.Eventually(t, func() bool {
			results := s.monitor.Results()
			_, okCPU := results["system_cpu"]
			_, okMem := results["system_mem"]
			_, okDiskRoot := results["disk_root"]
			_, okDiskTmp := results["disk_tmp"]
			return okCPU && okMem && okDiskRoot && okDiskTmp
		}, 500*time.Millisecond, 10*time.Millisecond, "monitor results should contain system and disk items")
		// Note: Health checks and pings may not appear immediately in results without actual network calls
	})

	t.Run("error_invalid_disk_config", func(t *testing.T) {
		cfg := config.MonitoringConfig{
			Interval: 30,
			System: config.SystemConfig{
				CPU:    config.SystemItem{Threshold: 80},
				Memory: config.SystemItem{Threshold: 85},
			},
			Disk: map[string]config.DiskConfig{
				"invalid": {Path: "/nonexistent/path/that/should/not/exist", Threshold: 90},
			},
		}

		// This should still succeed as NewDisk doesn't validate path existence
		err := s.SetConfig(ctx, cfg)
		assert.NoError(t, err)
	})

	t.Run("success_with_invalid_healthcheck_url", func(t *testing.T) {
		cfg := config.MonitoringConfig{
			Interval: 30,
			System: config.SystemConfig{
				CPU:    config.SystemItem{Threshold: 80},
				Memory: config.SystemItem{Threshold: 85},
			},
			Healthcheck: map[string]config.HealthcheckConfig{
				"invalid": {URL: "not-a-valid-url", Timeout: 5000},
			},
		}

		// NewHealthcheck seems to accept invalid URLs and just fails at check time
		err := s.SetConfig(ctx, cfg)
		assert.NoError(t, err, "invalid URL still allows creation, fails at check time")
	})

	t.Run("success_with_empty_ping_address", func(t *testing.T) {
		cfg := config.MonitoringConfig{
			Interval: 30,
			System: config.SystemConfig{
				CPU:    config.SystemItem{Threshold: 80},
				Memory: config.SystemItem{Threshold: 85},
			},
			Ping: map[string]config.PingConfig{
				"invalid": {Address: "", AmberThreshold: 100}, // Empty address is allowed, fails at check time
			},
		}

		err := s.SetConfig(ctx, cfg)
		assert.NoError(t, err, "empty ping address is allowed, fails at check time")
	})

	t.Run("error_invalid_ping_config_missing_amber_threshold", func(t *testing.T) {
		cfg := config.MonitoringConfig{
			Interval: 30,
			System: config.SystemConfig{
				CPU:    config.SystemItem{Threshold: 80},
				Memory: config.SystemItem{Threshold: 85},
			},
			Ping: map[string]config.PingConfig{
				"invalid": {Address: "8.8.8.8"}, // Missing AmberThreshold should cause error
			},
		}

		err := s.SetConfig(ctx, cfg)
		assert.Error(t, err, "should fail with missing amber threshold")
	})

	t.Run("success_with_k8s_monitors", func(t *testing.T) {
		s.config.Kubernetes.Enabled = true
		cfg := config.MonitoringConfig{
			Interval: 30,
			System: config.SystemConfig{
				CPU:    config.SystemItem{Threshold: 80},
				Memory: config.SystemItem{Threshold: 85},
			},
			K8sEvents: map[string]config.K8sEventsConfig{
				"pod-events": {Namespaces: "default", Threshold: 5, Window: 300},
			},
			K8sNodes: map[string]config.K8sNodesConfig{
				"cluster": {Icon: "server"},
			},
			K8sService: map[string]config.K8sServiceConfig{
				"api": {Service: "api-svc", Namespace: "default", Threshold: 80, Port: 8080},
			},
		}

		err := s.SetConfig(ctx, cfg)
		assert.NoError(t, err)

		assert.Eventually(t, func() bool {
			results := s.monitor.Results()
			_, okEvents := results["k8sevents_pod-events"]
			_, okNodes := results["k8snodes_cluster"]
			_, okSvc := results["k8sservice_api"]
			return okEvents && okNodes && okSvc
		}, 500*time.Millisecond, 10*time.Millisecond, "k8s monitors should appear in results")
	})

	t.Run("k8s_monitors_skipped_when_disabled", func(t *testing.T) {
		s.config.Kubernetes.Enabled = false
		cfg := config.MonitoringConfig{
			Interval: 30,
			System: config.SystemConfig{
				CPU:    config.SystemItem{Threshold: 80},
				Memory: config.SystemItem{Threshold: 85},
			},
			K8sEvents: map[string]config.K8sEventsConfig{
				"should-not-exist": {Namespaces: "default", Threshold: 5},
			},
		}

		err := s.SetConfig(ctx, cfg)
		assert.NoError(t, err)

		time.Sleep(50 * time.Millisecond)
		results := s.monitor.Results()
		_, exists := results["k8sevents_should-not-exist"]
		assert.False(t, exists, "k8s monitor should not be added when kubernetes is disabled")
	})
}

// TestAddK8sEventsMonitor verifies adding a K8s events monitor via the API.
func TestAddK8sEventsMonitor(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	data := config.K8sEventsConfig{
		Namespaces:     "default",
		Threshold:      5,
		Window:         300,
		Icon:           "zap",
		AlertThreshold: 1,
	}
	id := "test-events"
	resp, body := PostTestRequest(ctx, t, s, "/api/config/k8sevents/"+id, data)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	cfg, ok := s.config.Monitoring.K8sEvents[id]
	assert.True(t, ok, "k8sevents entry exists")
	assert.Equal(t, data.Namespaces, cfg.Namespaces)
	assert.Equal(t, data.Threshold, cfg.Threshold)
}

// TestDeleteK8sEventsMonitor verifies deleting a K8s events monitor via the API.
func TestDeleteK8sEventsMonitor(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	data := config.K8sEventsConfig{Namespaces: "default", Threshold: 5}
	id := "test-events"
	resp, _ := PostTestRequest(ctx, t, s, "/api/config/k8sevents/"+id, data)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, body := DeleteTestRequest(ctx, t, s, "/api/config/k8sevents/"+id)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "OK\n", body)
	assert.Equal(t, 0, len(s.config.Monitoring.K8sEvents))
}

// TestAddK8sNodesMonitor verifies adding a K8s nodes monitor via the API.
func TestAddK8sNodesMonitor(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	data := config.K8sNodesConfig{
		Icon:           "server",
		AlertThreshold: 2,
	}
	id := "test-nodes"
	resp, body := PostTestRequest(ctx, t, s, "/api/config/k8snodes/"+id, data)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	cfg, ok := s.config.Monitoring.K8sNodes[id]
	assert.True(t, ok, "k8snodes entry exists")
	assert.Equal(t, data, cfg, "k8snodes entry applied")
}

// TestDeleteK8sNodesMonitor verifies deleting a K8s nodes monitor via the API.
func TestDeleteK8sNodesMonitor(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	data := config.K8sNodesConfig{Icon: "server"}
	id := "test-nodes"
	resp, _ := PostTestRequest(ctx, t, s, "/api/config/k8snodes/"+id, data)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, body := DeleteTestRequest(ctx, t, s, "/api/config/k8snodes/"+id)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "OK\n", body)
	assert.Equal(t, 0, len(s.config.Monitoring.K8sNodes))
}

// TestAddK8sServiceMonitor verifies adding a K8s service monitor via the API.
func TestAddK8sServiceMonitor(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	data := config.K8sServiceConfig{
		Namespace:      "default",
		Service:        "api-svc",
		HealthPath:     "/healthz",
		Port:           8080,
		Threshold:      80,
		Icon:           "globe",
		AlertThreshold: 1,
	}
	id := "test-service"
	resp, body := PostTestRequest(ctx, t, s, "/api/config/k8sservice/"+id, data)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	cfg, ok := s.config.Monitoring.K8sService[id]
	assert.True(t, ok, "k8sservice entry exists")
	assert.Equal(t, data.Service, cfg.Service)
	assert.Equal(t, data.Threshold, cfg.Threshold)
}

// TestDeleteK8sServiceMonitor verifies deleting a K8s service monitor via the API.
func TestDeleteK8sServiceMonitor(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	data := config.K8sServiceConfig{Service: "svc", Threshold: 80}
	id := "test-service"
	resp, _ := PostTestRequest(ctx, t, s, "/api/config/k8sservice/"+id, data)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, body := DeleteTestRequest(ctx, t, s, "/api/config/k8sservice/"+id)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "OK\n", body)
	assert.Equal(t, 0, len(s.config.Monitoring.K8sService))
}

// TestDeleteMonitorInvalidType verifies that deleting with an unknown type returns 400.
func TestDeleteMonitorInvalidType(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	resp, _ := DeleteTestRequest(ctx, t, s, "/api/config/invalidtype/some-id")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestAddDockerMonitor verifies adding a Docker monitor via the API.
func TestAddDockerMonitor(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	data := config.DockerConfig{
		Containers:     "nginx,redis",
		Threshold:      2,
		Icon:           "box",
		AlertThreshold: 1,
	}
	id := "test-docker"
	resp, body := PostTestRequest(ctx, t, s, "/api/config/docker/"+id, data)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	cfg, ok := s.config.Monitoring.Docker[id]
	assert.True(t, ok, "docker entry exists")
	assert.Contains(t, cfg.Containers, "nginx")
	assert.Contains(t, cfg.Containers, "redis")
	assert.Equal(t, data.Threshold, cfg.Threshold)
}

// TestDeleteDockerMonitor verifies deleting a Docker monitor via the API.
func TestDeleteDockerMonitor(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	data := config.DockerConfig{Containers: "nginx", Threshold: 1}
	id := "test-docker"
	resp, _ := PostTestRequest(ctx, t, s, "/api/config/docker/"+id, data)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, body := DeleteTestRequest(ctx, t, s, "/api/config/docker/"+id)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "OK\n", body)
	assert.Equal(t, 0, len(s.config.Monitoring.Docker))
}
