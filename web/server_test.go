// server_test.go contains integration and unit tests for the lmon web server.
// These tests cover HTTP endpoints, configuration management, static file serving, and webhook integration.
package web

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"lmon/config"
	"lmon/monitors"
	"lmon/monitors/disk"
	"lmon/monitors/healthcheck"
	"lmon/monitors/system"
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
	assert.True(t, within(len(indexHtml), len(body), .10), "index returned is about the same length as the template")

	r, body = GetTestRequest(ctx, t, s, "/index.html")

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
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	r, body := GetTestRequest(ctx, t, s, "/config")

	assert.Equal(t, http.StatusOK, r.StatusCode, "status code")
	assert.True(t, within(len(configHtml), len(body), .10), "config returned is about the same length as the template")
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
// TestDeleteDisk verifies deleting a disk monitor and handling repeated deletes (not found).
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
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "status code")
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
	err := json.Unmarshal([]byte(body), &stats)
	assert.NoError(t, err, "unmarshal")
	_, exists := stats["health_"+id]
	assert.False(t, exists, "deleted healthcheck should not be in /api/items")

	resp, body = DeleteTestRequest(ctx, t, s, "/api/config/health/"+id)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "status code")

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

//go:embed test/postsave.yaml
var expectedFile string

// TestConfigSaved verifies that configuration changes are persisted to disk.
func TestConfigSaved(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	dir := t.TempDir()
	testFile := strings.Join([]string{dir, "test.yaml"}, string(os.PathSeparator))

	s, _ := StartTestServer(ctx, t, testFile)
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

	bodyBytes, err := os.ReadFile(testFile)
	assert.NoError(t, err, "read config file")

	assert.Equal(t, expectedFile, string(bodyBytes), "config saved")
}

// TestMapper_SetConfig verifies that SetConfig applies all monitor types and persists configuration.
func TestMapper_SetConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")

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

	s, _ := StartTestServer(ctx, t, testFile)
	s.Start(ctx)

	resp, body := GetTestRequest(ctx, t, s, "/api/items")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")

	var stats map[string]monitors.Result
	err = json.Unmarshal([]byte(body), &stats)
	assert.NoError(t, err, "unmarshal")

	assert.Equal(t, 4, len(stats), "stats entries")

	t.Run("getItem cpu", func(t *testing.T) {
		resp, body = GetTestRequest(ctx, t, s, "/api/items/system/cpu")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	})
	t.Run("getItem mem", func(t *testing.T) {
		resp, body = GetTestRequest(ctx, t, s, "/api/items/system/mem")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")

	})
	t.Run("getItem disk", func(t *testing.T) {
		resp, body = GetTestRequest(ctx, t, s, "/api/items/disk/root")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")

	})
	t.Run("getItem health", func(t *testing.T) {
		resp, body = GetTestRequest(ctx, t, s, "/api/items/health/google")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	})
	t.Run("getItem missing", func(t *testing.T) {
		resp, body = GetTestRequest(ctx, t, s, "/api/items/disk/missing")
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

	s, _ := StartTestServer(ctx, t, testFile)
	s.Start(ctx)

	resp, body := GetTestRequest(ctx, t, s, "/api/config")
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
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	resp, _ := PostTestRequest(ctx, t, s, "/api/config/system", "not json")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "status code")

	resp, _ = PostTestRequest(ctx, t, s, "/api/config/interval", "not json")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "status code")
}

// TestDeleteBadType verifies that deleting a monitor with an invalid type returns HTTP 400 Bad Request.
func TestDeleteBadType(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	resp, _ := DeleteTestRequest(ctx, t, s, "/api/config/bad/123")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "status code")

}

func Test_nweUIResult(t *testing.T) {
	c := config.Config{
		Monitoring: config.MonitoringConfig{
			System: config.SystemConfig{
				CPU: config.SystemItem{
					Threshold: 90,
					Icon:      "cpu-icon",
				},
				Memory: config.SystemItem{
					Threshold: 90,
					Icon:      "mem-icon",
				},
			},
			Disk: map[string]config.DiskConfig{
				"test_d": {
					Threshold: 90,
					Icon:      "disk-icon",
				},
			},
			Healthcheck: map[string]config.HealthcheckConfig{
				"test_h": {
					URL:     "http://localhost:8080/healtz",
					Timeout: 1,
					Icon:    "health-icon",
				},
			},
		},
		Webhook: config.WebhookConfig{
			Enabled: true,
		},
	}
	type args struct {
		id   string
		item monitors.Result
	}
	tests := []struct {
		name string
		args args
		want UIResult
	}{
		{"disk", args{"disk_test_d", monitors.Result{Group: disk.Group, Status: monitors.RAGGreen}}, UIResult{Status: monitors.RAGGreen, Icon: "disk-icon", Group: disk.Group}},
		{"health", args{"health_test_h", monitors.Result{Group: healthcheck.Group, Status: monitors.RAGRed}}, UIResult{Status: monitors.RAGRed, Icon: "health-icon", Group: healthcheck.Group}},
		{"cpu", args{"system_cpu", monitors.Result{Group: system.Group, DisplayName: "cpu", Status: monitors.RAGRed}}, UIResult{Status: monitors.RAGRed, Icon: "cpu-icon", Group: system.Group, DisplayName: "cpu"}},
		{"mem", args{"system_mem", monitors.Result{Group: system.Group, DisplayName: "mem", Status: monitors.RAGAmber}}, UIResult{Status: monitors.RAGAmber, Icon: "mem-icon", Group: system.Group, DisplayName: "mem"}},
		{"disk-fallback", args{"disk_test_not-found", monitors.Result{Group: disk.Group, Status: monitors.RAGGreen}}, UIResult{Status: monitors.RAGGreen, Icon: disk.Icon, Group: disk.Group}},
		{"health-fallback", args{"health_test_not-found", monitors.Result{Group: healthcheck.Group, Status: monitors.RAGRed}}, UIResult{Status: monitors.RAGRed, Icon: healthcheck.Icon, Group: healthcheck.Group}},
		{"fallback", args{"unknown_unknown", monitors.Result{Group: "unknown", Status: monitors.RAGError}}, UIResult{Status: monitors.RAGError, Icon: "folder", Group: "unknown"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, nweUIResult(tt.args.id, tt.args.item, &c), "nweUIResult(%v, %v, %v)", tt.args.id, tt.args.item, c)
		})
	}
}
