package web

import (
	"context"
	_ "embed"
	"io"
	"log"
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
	"lmon/monitors/mapper"
	"lmon/monitors/system"
)

func NewMockImpplementations() *mapper.Implementations {
	return &mapper.Implementations{
		Disk:   disk.NewMockDiskProvider(50),
		Health: healthcheck.NewMockHealthcheckProvider(50),
		Cpu:    system.NewMockCpuProvider(50),
		Mem:    system.NewMockMemProvider(50),
	}
}

func startTestServer(ctx context.Context, t *testing.T, cfgFile string) *Server {
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
	s, err := NewServerWithContext(ctx, cfg, l, mon, mapper.NewBuilder(NewMockImpplementations()))
	require.NoError(t, err, "server create")
	return s
}

func TestNewServerWithContext_Smoke(t *testing.T) {
	defer goleak.VerifyNone(t)

	assert.NotPanics(t, func() {
		ctx, cancel := context.WithCancel(t.Context())
		s := startTestServer(ctx, t, "")
		err := s.Start()
		assert.NoError(t, err, "server start")

		time.Sleep(10 * time.Millisecond)

		cancel()
	})

	time.Sleep(10 * time.Millisecond)
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

func TestSelfHealthcheck(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s := startTestServer(ctx, t, "")
	err := s.Start()
	assert.NoError(t, err, "start")

	r, body := getRequest(ctx, t, s, "/healthz")

	assert.Equal(t, http.StatusOK, r.StatusCode, "status code")
	assert.Equal(t, "OK\n", body)
}

func TestGetIndex(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s := startTestServer(ctx, t, "")
	err := s.Start()
	assert.NoError(t, err, "start")

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
	s := startTestServer(ctx, t, "")
	err := s.Start()
	assert.NoError(t, err, "start")

	r, body := getRequest(ctx, t, s, "/config")

	assert.Equal(t, http.StatusOK, r.StatusCode, "status code")
	assert.True(t, within(len(configHtml), len(body), .10), "config returned is about the same length as the template")
}

//go:embed static/icon.svg
var icon string

func TestStatic(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s := startTestServer(ctx, t, "")
	err := s.Start()
	assert.NoError(t, err, "start")

	r, body := getRequest(ctx, t, s, "/static/icon.svg")
	assert.Equal(t, http.StatusOK, r.StatusCode, "status code")
	assert.Equal(t, icon, body)
}
