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

	// if config is requested, copy it to the temp folder
	if cfgFile != "" {
		c, err := os.ReadFile(cfgFile)
		require.NoError(t, err, "open config file")
		dest := strings.Join([]string{t.TempDir(), "config.yaml"}, string(os.PathSeparator))
		err = os.WriteFile(dest, c, 0644)

	}
	l := config.NewLoader("config.yaml", []string{t.TempDir()})
	cfg, err := l.Load()
	assert.NoError(t, err, "config loaded")
	push := monitors.NewMockPush()
	mon := monitors.NewService(ctx, 10*time.Millisecond, 10*time.Millisecond, push.Push)
	hook := &mockWebhookHandler{}
	s, err := NewServerWithContext(ctx, cfg, l, mon, mapper.NewBuilder(NewMockImplementations(hook)))
	require.NoError(t, err, "server create")
	return s, hook
}

func getRequest(ctx context.Context, t *testing.T, s *Server, path string) (*http.Response, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", s.serverUrl+path, nil)
	require.NoErrorf(t, err, "GET %s", s.serverUrl+path)

	client := http.Client{}
	res, err := client.Do(req)
	require.NoErrorf(t, err, "GET %s", s.serverUrl+path)

	return readBody(res, t)
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

func postRequest(ctx context.Context, t *testing.T, s *Server, path string, data any) (*http.Response, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	bodybyes, err := json.Marshal(data)
	require.NoError(t, err, "marshal data")
	bodyBuff := bytes.NewBuffer(bodybyes)

	req, err := http.NewRequestWithContext(ctx, "POST", s.serverUrl+path, bodyBuff)
	require.NoErrorf(t, err, "POST %s", s.serverUrl+path)

	client := http.Client{}
	res, err := client.Do(req)
	require.NoErrorf(t, err, "POST %s", s.serverUrl+path)

	return readBody(res, t)
}

func TestNewServerWithContext_Smoke(t *testing.T) {
	defer goleak.VerifyNone(t)

	assert.NotPanics(t, func() {
		ctx, cancel := context.WithCancel(t.Context())
		s, _ := startTestServer(ctx, t, "")
		err := s.Start()
		assert.NoError(t, err, "server start")

		time.Sleep(10 * time.Millisecond)

		cancel()
	})

	time.Sleep(10 * time.Millisecond)
}

func TestSelfHealthcheck(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := startTestServer(ctx, t, "")
	err := s.Start()
	require.NoError(t, err, "start")

	r, body := getRequest(ctx, t, s, "/healthz")

	assert.Equal(t, http.StatusOK, r.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)
}

func TestGetIndex(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := startTestServer(ctx, t, "")
	err := s.Start()
	require.NoError(t, err, "start")

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
	err := s.Start()
	require.NoError(t, err, "start")

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
	err := s.Start()
	require.NoError(t, err, "start")

	r, body := getRequest(ctx, t, s, "/static/icon.svg")
	assert.Equal(t, http.StatusOK, r.StatusCode, "status code")
	assert.Equal(t, icon, body)
}

func TestSetSystemConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := startTestServer(ctx, t, "")
	err := s.Start()
	require.NoError(t, err, "start")

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
	err := s.Start()
	require.NoError(t, err, "start")

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
	err := s.Start()
	require.NoError(t, err, "start")

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

func TestAddHealthcheck(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := startTestServer(ctx, t, "")
	err := s.Start()
	require.NoError(t, err, "start")

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

func TestSetWebhook(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := startTestServer(ctx, t, "")
	err := s.Start()
	require.NoError(t, err, "start")

	data := config.WebhookConfig{
		Enabled: true,
		URL:     s.serverUrl + "/hook",
	}

	resp, body := postRequest(ctx, t, s, "/api/config/testhook", data)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)

	assert.Equal(t, data, s.config.Webhook, "healthcheck entry applied")
}

func TestWebHookAndCallback(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, hook := startTestServer(ctx, t, "")
	err := s.Start()
	require.NoError(t, err, "start")

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
