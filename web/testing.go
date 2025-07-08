package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"lmon/config"
	"lmon/monitors"
	"lmon/monitors/mapper"
)

// StartTestServer creates and starts a new test server instance with a temporary configuration.
// Returns the server and a mock webhook handler for assertions.
func StartTestServer(ctx context.Context, t *testing.T, cfgFile string) (*Server, *MockWebhookHandler) {
	t.Helper()
	t.Setenv("LMON_WEB_PORT", "0")
	t.Setenv("LMON_WEB_HOST", "127.0.0.1")

	l := config.NewLoader(cfgFile, []string{t.TempDir()})
	cfg, err := l.Load()
	assert.NoError(t, err, "config loaded")
	push := monitors.NewMockPush()
	mon := monitors.NewService(ctx, 10*time.Millisecond, 10*time.Millisecond, push.Push)
	hook := &MockWebhookHandler{}
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
		req, err = http.NewRequestWithContext(ctx, method, s.ServerUrl+path, bodyBuff)
	} else {
		req, err = http.NewRequestWithContext(ctx, method, s.ServerUrl+path, nil)
	}

	require.NoErrorf(t, err, "GET %s", s.ServerUrl+path)

	client := http.Client{}
	res, err := client.Do(req)
	require.NoErrorf(t, err, "GET %s", s.ServerUrl+path)

	return readBody(res, t)
}

// GetTestRequest sends a GET request to the test server.
func GetTestRequest(ctx context.Context, t *testing.T, s *Server, path string) (*http.Response, string) {
	return sendRequest(ctx, t, "GET", s, path, 20*time.Millisecond, nil)
}

// DeleteTestRequest sends a DELETE request to the test server.
func DeleteTestRequest(ctx context.Context, t *testing.T, s *Server, path string) (*http.Response, string) {
	return sendRequest(ctx, t, "DELETE", s, path, 10*time.Millisecond, nil)
}

// PostTestRequest sends a POST request with JSON data to the test server.
func PostTestRequest(ctx context.Context, t *testing.T, s *Server, path string, data any) (*http.Response, string) {
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
