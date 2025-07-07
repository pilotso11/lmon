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

type mockWebhookHandler struct {
	lastMessage atomic.String
	cnt         atomic.Int32
}

func (m *mockWebhookHandler) webhookCallback(msg string) {
	m.lastMessage.Store(msg)
	m.cnt.Inc()
}

func NewMockImplementations(hook *mockWebhookHandler) *mapper.Implementations {
	return &mapper.Implementations{
		Disk:    disk.NewMockDiskProvider(50),
		Health:  healthcheck.NewMockHealthcheckProvider(50),
		Cpu:     system.NewMockCpuProvider(50),
		Mem:     system.NewMockMemProvider(50),
		Webhook: hook.webhookCallback,
	}
}

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

func getRequest(ctx context.Context, t *testing.T, s *Server, path string) (*http.Response, string) {
	return sendRequest(ctx, t, "GET", s, path, 20*time.Millisecond, nil)
}

func deleteRequest(ctx context.Context, t *testing.T, s *Server, path string) (*http.Response, string) {
	return sendRequest(ctx, t, "DELETE", s, path, 10*time.Millisecond, nil)
}

func postRequest(ctx context.Context, t *testing.T, s *Server, path string, data any) (*http.Response, string) {
	return sendRequest(ctx, t, "POST", s, path, 100*time.Millisecond, data)
}

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

func TestSelfHealthcheck(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := startTestServer(ctx, t, "")
	s.Start()

	r, body := getRequest(ctx, t, s, "/healthz")

	assert.Equal(t, http.StatusOK, r.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)
}

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

func within(i int, i2 int, tolerance float64) bool {
	d := i - i2
	if d < 0 {
		d = -d
	}
	return float64(d)/float64(i) < tolerance
}

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

func TestStatic(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := startTestServer(ctx, t, "")
	s.Start()

	r, body := getRequest(ctx, t, s, "/static/icon.svg")
	assert.Equal(t, http.StatusOK, r.StatusCode, "status code")
	assert.Equal(t, icon, body)
}

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

	resp, body = deleteRequest(ctx, t, s, "/api/config/disk/"+id)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "status code")
}

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

	assert.Equal(t, 0, len(s.config.Monitoring.Disk), "number of disk entries")

	resp, body = deleteRequest(ctx, t, s, "/api/config/healthcheck/"+id)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "status code")

}

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

func TestDeleteBadType(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := startTestServer(ctx, t, "")
	s.Start()

	resp, _ := deleteRequest(ctx, t, s, "/api/config/bad/123")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "status code")

}
