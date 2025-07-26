// health_test.go contains unit tests for the Healthcheck monitor implementation and its integration with the monitoring service.
package healthcheck

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"

	"lmon/common"
	"lmon/config"
	"lmon/monitors"
)

// TestNewHealthcheck verifies that a Healthcheck can be created and added to a monitoring service.
func TestNewHealthcheck(t *testing.T) {
	push := monitors.NewMockPush()
	h, err := NewHealthcheck("local", "http://localhost/health", 5, 0, "", MockHealthcheckProvider{Result: atomic.NewInt32(http.StatusOK)})
	assert.NoError(t, err)
	svc := monitors.NewService(t.Context(), time.Second, time.Millisecond, push.Push)
	svc.Add(t.Context(), h)
	assert.Equal(t, 1, svc.Size(), "one monitor added")
}

// TestHealthcheck_DisplayName verifies that DisplayName returns the expected string for various healthcheck names and URLs.
func TestHealthcheck_DisplayName(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		timeout  int
		respCode int
		icon     string
		expect   string
		wantErr  bool
	}{
		{"test", "http://test", 5, 0, "", "test (http://test)", false},
		{"test", "http://test", 5, 401, "", "test (http://test - 401)", false},
		{"test path", "http://test/long", 5, 0, "icon", "test path (http://test)", false},
		{"test long server", "http://test.test.com/long", 5, 0, "icon", "test long server (http://test.test.com)", false},
		{"test port", "https://test.test.com:1234/long", 5, 0, "", "test port (https://test.test.com:1234)", false},
		{"fail", "http://\u0012bad/5", 5, 0, "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewHealthcheck(tt.name, tt.url, tt.timeout, tt.respCode, tt.icon, nil)
			if tt.wantErr {
				assert.Error(t, err, "error expected")
			} else {
				assert.NoError(t, err, "error not expected")
				assert.Equal(t, tt.expect, c.DisplayName(), "DisplayName()")
			}
		})
	}
}

// TestHealthcheck_Group verifies that Group returns the correct group name for healthcheck monitors.
func TestHealthcheck_Group(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		timeout  int
		respCode int
		icon     string
		expect   string
		wantErr  bool
	}{
		{"test", "http://test", 5, 0, "", "health", false},
		{"test path", "http://test/long", 5, 0, "icon", "health", false},
		{"test long server", "http://test.test.com/long", 5, 0, "icon", "health", false},
		{"test port", "https://test.test.com:1234/long", 5, 0, "", "health", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewHealthcheck(tt.name, tt.url, tt.timeout, tt.respCode, tt.icon, nil)
			if tt.wantErr {
				assert.Error(t, err, "error expected")
			} else {
				assert.NoError(t, err, "error not expected")
				assert.Equal(t, tt.expect, c.Group(), "Group()")
			}
		})
	}
}

// TestHealthcheck_Name verifies that Name returns the correct unique identifier for healthcheck monitors.
func TestHealthcheck_Name(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		timeout  int
		respCode int
		icon     string
		expect   string
		wantErr  bool
	}{
		{"test", "http://test", 5, 0, "", "health_test", false},
		{"test path", "http://test/long", 5, 0, "icon", "health_test path", false},
		{"test long server", "http://test.test.com/long", 5, 401, "icon", "health_test long server", false},
		{"test port", "https://test.test.com:1234/long", 5, 0, "", "health_test port", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewHealthcheck(tt.name, tt.url, tt.timeout, tt.respCode, tt.icon, nil)
			if tt.wantErr {
				assert.Error(t, err, "error expected")
			} else {
				assert.NoError(t, err, "error not expected")
				assert.Equal(t, tt.expect, c.Name(), "Name()")
			}
		})
	}

}

// TestHealthcheck_Save verifies that Save correctly persists the healthcheck monitor configuration.
func TestHealthcheck_Save(t *testing.T) {
	l := config.NewLoader("", []string{t.TempDir()})
	cfg, _ := l.Load()

	// Arrange
	c, err := NewHealthcheck("test", "http://localhost/health", 5, 401, "icon-test", nil)
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
	assert.Equal(t, 401, h.RespCode, "respCode")
}

// TestHealthcheck_DefaultImplSmokeTest verifies the default implementation of Healthcheck.Check
// for various HTTP status codes and simulated delays.
func TestHealthcheck_DefaultImplSmokeTest(t *testing.T) {
	ts := common.StartTestServer(t, "/health")
	defer func() {
		_ = ts.Server.Shutdown(t.Context())
	}()

	tests := []struct {
		name    string
		expCode int
		resp    int32
		msDelay int32
		expect  monitors.RAG
	}{
		{"200", 0, http.StatusOK, 0, monitors.RAGGreen},
		{"500", 0, http.StatusInternalServerError, 0, monitors.RAGRed},
		{"307 Redirected", 0, http.StatusTemporaryRedirect, 0, monitors.RAGAmber},
		{"404 Not found", 0, http.StatusNotFound, 0, monitors.RAGAmber},
		{"401 Unauthorized", 0, http.StatusUnauthorized, 0, monitors.RAGAmber},
		{"200 timeout", 0, http.StatusOK, 2000, monitors.RAGError},
		{"200 want 401", 401, http.StatusOK, 0, monitors.RAGGreen},
		{"401 want 401", 401, http.StatusUnauthorized, 0, monitors.RAGGreen},
		{"404 want 401", 401, http.StatusNotFound, 0, monitors.RAGAmber},
		{"307 want 401", 401, http.StatusTemporaryRedirect, 0, monitors.RAGAmber},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts.RespCode.Store(tt.resp)
			ts.DelayMs.Store(tt.msDelay)
			c, err := NewHealthcheck("test", ts.URL, 1, tt.expCode, "", nil)
			assert.NoError(t, err)
			r := c.Check(t.Context())
			assert.Equal(t, tt.expect, r.Status, "status not %s: %s", tt.expect.String(), r.Status.String())

		})
	}
}
