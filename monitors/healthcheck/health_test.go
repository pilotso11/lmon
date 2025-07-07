package healthcheck

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"

	"lmon/config"
	"lmon/monitors"
)

type MockHealthcheckProvider struct {
	result *atomic.Int32
	err    error
}

func (m MockHealthcheckProvider) Check(_ context.Context, _ *url.URL, _ int) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &http.Response{
		StatusCode: int(m.result.Load()),
		Status:     http.StatusText(int(m.result.Load())),
	}, nil
}

func TestNewHealthcheck(t *testing.T) {
	push := monitors.NewMockPush()
	h, err := NewHealthcheck("local", "http://localhost/health", 5, "", MockHealthcheckProvider{result: atomic.NewInt32(http.StatusOK)})
	assert.NoError(t, err)
	svc := monitors.NewService(t.Context(), time.Second, time.Millisecond, push.Push)
	_ = svc.Add(t.Context(), h)
	assert.Equal(t, 1, len(svc.Monitors), "one monitor added")
}

func TestHealthcheck_DisplayName(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		timeout int
		icon    string
		expect  string
		wantErr bool
	}{
		{"test", "http://test", 5, "", "test (http://test)", false},
		{"test path", "http://test/long", 5, "icon", "test path (http://test)", false},
		{"test long server", "http://test.test.com/long", 5, "icon", "test long server (http://test.test.com)", false},
		{"test port", "https://test.test.com:1234/long", 5, "", "test port (https://test.test.com:1234)", false},
		{"fail", "http://\u0012bad/5", 5, "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewHealthcheck(tt.name, tt.url, tt.timeout, tt.icon, nil)
			if tt.wantErr {
				assert.Error(t, err, "error expected")
			} else {
				assert.NoError(t, err, "error not expected")
				assert.Equal(t, tt.expect, c.DisplayName(), "DisplayName()")
			}
		})
	}
}

func TestHealthcheck_Group(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		timeout int
		icon    string
		expect  string
		wantErr bool
	}{
		{"test", "http://test", 5, "", "app", false},
		{"test path", "http://test/long", 5, "icon", "app", false},
		{"test long server", "http://test.test.com/long", 5, "icon", "app", false},
		{"test port", "https://test.test.com:1234/long", 5, "", "app", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewHealthcheck(tt.name, tt.url, tt.timeout, tt.icon, nil)
			if tt.wantErr {
				assert.Error(t, err, "error expected")
			} else {
				assert.NoError(t, err, "error not expected")
				assert.Equal(t, tt.expect, c.Group(), "Group()")
			}
		})
	}
}

func TestHealthcheck_Name(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		timeout int
		icon    string
		expect  string
		wantErr bool
	}{
		{"test", "http://test", 5, "", "healthcheck_test", false},
		{"test path", "http://test/long", 5, "icon", "healthcheck_test path", false},
		{"test long server", "http://test.test.com/long", 5, "icon", "healthcheck_test long server", false},
		{"test port", "https://test.test.com:1234/long", 5, "", "healthcheck_test port", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewHealthcheck(tt.name, tt.url, tt.timeout, tt.icon, nil)
			if tt.wantErr {
				assert.Error(t, err, "error expected")
			} else {
				assert.NoError(t, err, "error not expected")
				assert.Equal(t, tt.expect, c.Name(), "Name()")
			}
		})
	}

}

func TestHealthcheck_Save(t *testing.T) {
	l := config.NewLoader("", []string{t.TempDir()})
	cfg, _ := l.Load()

	// Arrange
	c, err := NewHealthcheck("test", "http://localhost/health", 5, "icon-test", nil)
	assert.NoError(t, err)
	// Act
	c.Save(cfg)

	// Assert
	assert.Equal(t, 1, len(cfg.Monitoring.Healthcheck), "len(Healthcheck)")
	h, ok := cfg.Monitoring.Healthcheck["test"]
	assert.True(t, ok, "found name")
	assert.Equal(t, "http://localhost/health", h.URL, "URL")
	assert.Equal(t, "icon-test", h.Icon, "icon")
	assert.Equal(t, 5, h.Timeout, "timeout")
}

type testServer struct {
	ts      *http.Server
	code    atomic.Int32
	msDelay atomic.Int32
	url     string
}

func (ts *testServer) handler(w http.ResponseWriter, _ *http.Request) {
	delay := ts.msDelay.Load()
	code := int(ts.code.Load())
	if delay > 0 {
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
	if code == http.StatusOK {
		w.WriteHeader(code)
		_, _ = w.Write([]byte(http.StatusText(code)))
	} else {
		http.Error(w, http.StatusText(code), code)
	}

}

func startTestServer(t *testing.T, uri string) *testServer {
	ts := &testServer{}
	ts.ts = &http.Server{}
	mux := http.NewServeMux()
	mux.HandleFunc(uri, ts.handler)
	ts.ts.Handler = mux

	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	assert.NoError(t, err)
	ts.url = "http://" + ln.Addr().String() + "/health"
	go func() {
		_ = ts.ts.Serve(ln)
	}()

	return ts
}

func TestHealthcheck_DefaultImplSmokeTest(t *testing.T) {
	ts := startTestServer(t, "/health")
	defer func() {
		_ = ts.ts.Shutdown(t.Context())
	}()

	tests := []struct {
		name    string
		resp    int32
		msDelay int32
		expect  monitors.RAG
	}{
		{"200", http.StatusOK, 0, monitors.RAGGreen},
		{"500", http.StatusInternalServerError, 0, monitors.RAGRed},
		{"307 Redirected", http.StatusTemporaryRedirect, 0, monitors.RAGAmber},
		{"404 Not found", http.StatusNotFound, 0, monitors.RAGAmber},
		{"200 timeout", http.StatusOK, 100, monitors.RAGError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts.code.Store(tt.resp)
			ts.msDelay.Store(tt.msDelay)
			c, err := NewHealthcheck("test", ts.url, 20, "", nil)
			assert.NoError(t, err)
			r := c.Check(t.Context())
			assert.Equal(t, tt.expect, r.Status, "status not %s: %s", tt.expect.String(), r.Status.String())

		})
	}
}
