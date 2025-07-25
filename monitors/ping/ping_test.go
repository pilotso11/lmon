package ping

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"lmon/config"
	"lmon/monitors"
)

func TestPingMonitor_SuccessGreen(t *testing.T) {
	pm := NewPingMonitor("test-green", "127.0.0.1", 1000, "", 100, NewMockPingProvider(50, nil))
	result := pm.Check(t.Context())
	if result.Status != monitors.RAGGreen {
		t.Errorf("Expected RAGGreen, got %v", result.Status)
	}
	if result.Value != "50 ms" {
		t.Errorf("Expected value '50 ms', got %s", result.Value)
	}
}

func TestPingMonitor_SuccessAmber(t *testing.T) {
	pm := NewPingMonitor("test-amber", "127.0.0.1", 1000, "", 100, NewMockPingProvider(150, nil))
	result := pm.Check(t.Context())
	if result.Status != monitors.RAGAmber {
		t.Errorf("Expected RAGAmber, got %v", result.Status)
	}
	if result.Value != "150 ms" {
		t.Errorf("Expected value '150 ms', got %s", result.Value)
	}
}

func TestPingMonitor_FailureRed(t *testing.T) {
	pm := NewPingMonitor("test-red", "127.0.0.1", 1000, "", 100, NewMockPingProvider(500, errors.New("timeout")))
	result := pm.Check(t.Context())
	assert.Equal(t, monitors.RAGRed.String(), result.Status.String(), "Status is red")
	if result.Value == "" || result.Value == "0 ms" {
		t.Errorf("Expected error message, got %s", result.Value)
	}
}

func TestPingMonitor_DisplayNameAndGroup(t *testing.T) {
	pm := NewPingMonitor("display", "localhost", 1000, "", 100, NewMockPingProvider(10, nil))
	assert.Equal(t, "Ping: display (localhost)", pm.DisplayName(), "DisplayName")
	assert.Equal(t, "ping", pm.Group(), "Group")
	assert.Equal(t, "ping_display", pm.Name(), "Name")
}

func TestPingMonitor_Save(t *testing.T) {
	cfg := &config.Config{}
	pm := NewPingMonitor("save-test", "1.2.3.4", 1234, "wifi", 200, NewMockPingProvider(10, nil))
	pm.Save(cfg)
	pc, ok := cfg.Monitoring.Ping["save-test"]
	if !ok {
		t.Errorf("Ping config not saved")
	}
	if pc.Address != "1.2.3.4" || pc.Timeout != 1234 || pc.Icon != "wifi" || pc.AmberThreshold != 200 {
		t.Errorf("Ping config values incorrect: %+v", pc)
	}
}

func TestPingMonitor_DefaultProvider(t *testing.T) {
	pm := NewPingMonitor("default-provider", "localhost", 1000, "", 100, nil)
	if pm.impl == nil {
		t.Errorf("Default provider not set")
	}
}

func TestDefaultPingProvider_Ping_Error(t *testing.T) {
	provider := &DefaultPingProvider{}
	ctx := t.Context()
	_, err := provider.Ping(ctx, "invalid.invalid", 100)
	if err == nil {
		t.Errorf("Expected error for invalid ping address")
	}
}

func TestPingMonitor_Save_NilConfig(t *testing.T) {
	pm := NewPingMonitor("nil-test", "127.0.0.1", 1000, "", 100, NewMockPingProvider(10, nil))
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when saving to nil config")
		}
	}()
	pm.Save(nil)
}

func TestPingMonitor_FallbackElapsedTime(t *testing.T) {
	pm := NewPingMonitor("fallback-elapsed", "127.0.0.1", 1000, "", 100, NewMockPingProvider(0, nil))
	result := pm.Check(t.Context())
	if result.Value != "0 ms" {
		t.Errorf("Expected fallback value '0 ms', got %s", result.Value)
	}
}

func TestDefaultPingProvider_Ping_Localhost(t *testing.T) {
	// Skip in short mode or CI environments where ICMP is typically blocked
	if testing.Short() {
		t.Skip("Skipping ping test in short mode")
	}

	// Check for common CI environment variables
	ciVars := []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "TRAVIS", "CIRCLECI", "JENKINS_URL"}
	for _, env := range ciVars {
		if os.Getenv(env) != "" {
			t.Skipf("Skipping ping test in CI environment (detected %s)", env)
		}
	}

	provider := &DefaultPingProvider{}
	ctx := t.Context()
	ms, err := provider.Ping(ctx, "127.0.0.1", 1000)
	if err != nil {
		// Final fallback: skip if ping fails (may still be restricted environment)
		t.Skipf("Ping to localhost failed (restricted environment): %v", err)
	}
	if ms < 0 {
		t.Errorf("Ping response time should be >= 0, got %d", ms)
	}
}

func TestDefaultPingProvider_Ping_Unreachable(t *testing.T) {
	provider := &DefaultPingProvider{}
	ctx := t.Context()
	// 203.0.113.0 is a TEST-NET-3 address, reserved for documentation and should not reply to ICMP echo
	_, err := provider.Ping(ctx, "203.0.113.0", 1000)
	if err == nil {
		t.Errorf("Expected error for unreachable TEST-NET-3 address")
	}
}

func TestDefaultAmberValue(t *testing.T) {
	pm := NewPingMonitor("amber-value-test", "localhost", 100, Icon, 0, &DefaultPingProvider{})
	assert.Equal(t, 50, pm.amberThreshold, "Default amber threshold should be 50 ms")
}
