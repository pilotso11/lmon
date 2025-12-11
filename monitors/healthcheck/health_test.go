// health_test.go contains unit tests for the Healthcheck monitor implementation and its integration with the monitoring service.
package healthcheck

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"

	"lmon/common"
	"lmon/config"
	"lmon/monitors"
	"lmon/monitors/docker"
)

// TestNewHealthcheck verifies that a Healthcheck can be created and added to a monitoring service.
func TestNewHealthcheck(t *testing.T) {
	push := monitors.NewMockPush()
	h, err := NewHealthcheck("local", "http://localhost/health", 5, 0, "", "", 0, nil, MockHealthcheckProvider{Result: atomic.NewInt32(http.StatusOK)}, nil)
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
			c, err := NewHealthcheck(tt.name, tt.url, tt.timeout, tt.respCode, tt.icon, "", 0, nil, nil, nil)
			if tt.wantErr {
				assert.Error(t, err, "error expected")
			} else {
				assert.NoError(t, err, "error not expected")
				assert.Equal(t, tt.expect, c.DisplayName(), "DisplayName()")
			}
		})
	}
}

// TestHealthcheck_HasRestartContainers verifies the HasRestartContainers method.
func TestHealthcheck_HasRestartContainers(t *testing.T) {
	tests := []struct {
		name              string
		restartContainers string
		expect            bool
	}{
		{"no containers", "", false},
		{"single container", "myapp", true},
		{"multiple containers", "app1,app2", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := NewHealthcheck("test", "http://localhost", 5, 0, "", tt.restartContainers, 0, nil, nil, nil)
			assert.NoError(t, err)
			assert.Equal(t, tt.expect, h.HasRestartContainers())
		})
	}
}

// TestHealthcheck_ParseContainerList verifies the parseContainerList function.
func TestParseContainerList(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect []string
	}{
		{"empty", "", []string{}},
		{"single", "app1", []string{"app1"}},
		{"comma separated", "app1,app2,app3", []string{"app1", "app2", "app3"}},
		{"extra whitespace", "  app1 ,  app2  , app3  ", []string{"app1", "app2", "app3"}},
		{"trailing comma", "app1,app2,", []string{"app1", "app2"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseContainerList(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}

// TestHealthcheck_Save_WithRestartContainers verifies Save includes RestartContainers.
func TestHealthcheck_Save_WithRestartContainers(t *testing.T) {
	h, err := NewHealthcheck("test", "http://localhost", 5, 200, "icon", "app1,app2", 0, nil, nil, nil)
	assert.NoError(t, err)

	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			Healthcheck: make(map[string]config.HealthcheckConfig),
		},
	}
	h.Save(cfg)

	healthCfg, ok := cfg.Monitoring.Healthcheck["test"]
	assert.True(t, ok, "should save healthcheck config")
	assert.Equal(t, "app1,app2", healthCfg.RestartContainers)
	assert.Equal(t, "http://localhost", healthCfg.URL)
	assert.Equal(t, 5, healthCfg.Timeout)
	assert.Equal(t, 200, healthCfg.RespCode)
	assert.Equal(t, "icon", healthCfg.Icon)
}

// TestHealthcheck_Save_WithoutRestartContainers verifies Save handles empty RestartContainers.
func TestHealthcheck_Save_WithoutRestartContainers(t *testing.T) {
	h, err := NewHealthcheck("test", "http://localhost", 5, 0, "", "", 0, nil, nil, nil)
	assert.NoError(t, err)

	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			Healthcheck: make(map[string]config.HealthcheckConfig),
		},
	}
	h.Save(cfg)

	healthCfg, ok := cfg.Monitoring.Healthcheck["test"]
	assert.True(t, ok, "should save healthcheck config")
	assert.Equal(t, "", healthCfg.RestartContainers)
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
			c, err := NewHealthcheck(tt.name, tt.url, tt.timeout, tt.respCode, tt.icon, "", 0, nil, nil, nil)
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
			c, err := NewHealthcheck(tt.name, tt.url, tt.timeout, tt.respCode, tt.icon, "", 0, nil, nil, nil)
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
	c, err := NewHealthcheck("test", "http://localhost/health", 5, 401, "icon-test", "", 0, nil, nil, nil)
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
			c, err := NewHealthcheck("test", ts.URL, 1, tt.expCode, "", "", 0, nil, nil, nil)
			assert.NoError(t, err)
			r := c.Check(t.Context())
			assert.Equal(t, tt.expect, r.Status, "status not %s: %s", tt.expect.String(), r.Status.String())

		})
	}
}

// TestHealthcheck_RestartContainers_Success verifies successful container restart.
func TestHealthcheck_RestartContainers_Success(t *testing.T) {
	mockDocker := &docker.MockDockerProvider{}
	h, err := NewHealthcheck("test", "http://localhost", 5, 0, "", "app1,app2,app3", 0, nil, nil, mockDocker)
	assert.NoError(t, err)

	err = h.RestartContainers(t.Context())
	assert.NoError(t, err)

	// Verify the mock was called with correct containers
	assert.Equal(t, []string{"app1", "app2", "app3"}, mockDocker.RestartsRequested)
}

// TestHealthcheck_RestartContainers_Error verifies error handling.
func TestHealthcheck_RestartContainers_Error(t *testing.T) {
	mockDocker := &docker.MockDockerProvider{
		RestartError: assert.AnError,
	}
	h, err := NewHealthcheck("test", "http://localhost", 5, 0, "", "app1", 0, nil, nil, mockDocker)
	assert.NoError(t, err)

	err = h.RestartContainers(t.Context())
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

// TestHealthcheck_RestartContainers_NoContainers verifies error when no containers configured.
func TestHealthcheck_RestartContainers_NoContainers(t *testing.T) {
	h, err := NewHealthcheck("test", "http://localhost", 5, 0, "", "", 0, nil, nil, nil)
	assert.NoError(t, err)

	err = h.RestartContainers(t.Context())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no containers configured")
}

// TestHealthcheck_RestartContainers_NoProvider verifies error when provider not set.
func TestHealthcheck_RestartContainers_NoProvider(t *testing.T) {
	// Create healthcheck without docker provider but with containers configured
	h := Healthcheck{
		name:              "test",
		url:               &url.URL{Scheme: "http", Host: "localhost"},
		timeout:           5,
		respCode:          200,
		icon:              "activity",
		restartContainers: "app1",
		impl:              nil,
		dockerImpl:        nil,
	}

	err := h.RestartContainers(t.Context())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "docker provider not configured")
}
