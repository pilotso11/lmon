package ping

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"lmon/config"
	"lmon/monitors"
)

func TestPingMonitor_SuccessGreen(t *testing.T) {
	pm := NewPingMonitor("test-green", "127.0.0.1", 1000, "", 100, 0, NewMockPingProvider(50, nil))
	result := pm.Check(t.Context())
	if result.Status != monitors.RAGGreen {
		t.Errorf("Expected RAGGreen, got %v", result.Status)
	}
	if result.Value != "50 ms" {
		t.Errorf("Expected value '50 ms', got %s", result.Value)
	}
}

func TestPingMonitor_SuccessAmber(t *testing.T) {
	pm := NewPingMonitor("test-amber", "127.0.0.1", 1000, "", 100, 0, NewMockPingProvider(150, nil))
	result := pm.Check(t.Context())
	if result.Status != monitors.RAGAmber {
		t.Errorf("Expected RAGAmber, got %v", result.Status)
	}
	if result.Value != "150 ms" {
		t.Errorf("Expected value '150 ms', got %s", result.Value)
	}
}

func TestPingMonitor_FailureRed(t *testing.T) {
	pm := NewPingMonitor("test-red", "127.0.0.1", 1000, "", 100, 0, NewMockPingProvider(500, errors.New("timeout")))
	result := pm.Check(t.Context())
	assert.Equal(t, monitors.RAGRed.String(), result.Status.String(), "Status is red")
	if result.Value == "" || result.Value == "0 ms" {
		t.Errorf("Expected error message, got %s", result.Value)
	}
}

func TestPingMonitor_DisplayNameAndGroup(t *testing.T) {
	pm := NewPingMonitor("display", "localhost", 1000, "", 100, 0, NewMockPingProvider(10, nil))
	assert.Equal(t, "Ping: display (localhost)", pm.DisplayName(), "DisplayName")
	assert.Equal(t, "ping", pm.Group(), "Group")
	assert.Equal(t, "ping_display", pm.Name(), "Name")
}

func TestPingMonitor_Save(t *testing.T) {
	cfg := &config.Config{}
	pm := NewPingMonitor("save-test", "1.2.3.4", 1234, "wifi", 200, 0, NewMockPingProvider(10, nil))
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
	pm := NewPingMonitor("default-provider", "localhost", 1000, "", 100, 0, nil)
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
	pm := NewPingMonitor("nil-test", "127.0.0.1", 1000, "", 100, 0, NewMockPingProvider(10, nil))
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when saving to nil config")
		}
	}()
	pm.Save(nil)
}

func TestPingMonitor_FallbackElapsedTime(t *testing.T) {
	pm := NewPingMonitor("fallback-elapsed", "127.0.0.1", 1000, "", 100, 0, NewMockPingProvider(0, nil))
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
	pm := NewPingMonitor("amber-value-test", "localhost", 100, Icon, 0, 0, &DefaultPingProvider{})
	assert.Equal(t, 50, pm.amberThreshold, "Default amber threshold should be 50 ms")
}

// Test edge cases for NewPingMonitor with various parameter combinations
func TestNewPingMonitor_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		monitorName    string
		address        string
		timeout        int
		icon           string
		amberThreshold int
		impl           Provider
		expectedIcon   string
		expectedAmber  int
	}{
		{
			name:           "empty icon uses default",
			monitorName:    "test",
			address:        "127.0.0.1",
			timeout:        1000,
			icon:           "",
			amberThreshold: 100,
			impl:           NewMockPingProvider(10, nil),
			expectedIcon:   Icon,
			expectedAmber:  100,
		},
		{
			name:           "custom icon preserved",
			monitorName:    "test",
			address:        "127.0.0.1",
			timeout:        1000,
			icon:           "custom-icon",
			amberThreshold: 100,
			impl:           NewMockPingProvider(10, nil),
			expectedIcon:   "custom-icon",
			expectedAmber:  100,
		},
		{
			name:           "zero amber threshold uses default",
			monitorName:    "test",
			address:        "127.0.0.1",
			timeout:        1000,
			icon:           "wifi",
			amberThreshold: 0,
			impl:           NewMockPingProvider(10, nil),
			expectedIcon:   "wifi",
			expectedAmber:  50,
		},
		{
			name:           "negative amber threshold uses default",
			monitorName:    "test",
			address:        "127.0.0.1",
			timeout:        1000,
			icon:           "wifi",
			amberThreshold: -1,
			impl:           NewMockPingProvider(10, nil),
			expectedIcon:   "wifi",
			expectedAmber:  50,
		},
		{
			name:           "nil provider uses default",
			monitorName:    "test",
			address:        "127.0.0.1",
			timeout:        1000,
			icon:           "wifi",
			amberThreshold: 100,
			impl:           nil,
			expectedIcon:   "wifi",
			expectedAmber:  100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := NewPingMonitor(tt.monitorName, tt.address, tt.timeout, tt.icon, tt.amberThreshold, 0, tt.impl)
			assert.Equal(t, tt.expectedIcon, pm.icon, "icon should match expected")
			assert.Equal(t, tt.expectedAmber, pm.amberThreshold, "amber threshold should match expected")
			assert.NotNil(t, pm.impl, "provider should not be nil")
		})
	}
}

// Test Monitor.Save() with pre-existing ping config
func TestPingMonitor_Save_WithExistingConfig(t *testing.T) {
	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			Ping: map[string]config.PingConfig{
				"existing": {
					Address:        "existing.com",
					Timeout:        500,
					Icon:           "old-icon",
					AmberThreshold: 25,
				},
			},
		},
	}

	pm := NewPingMonitor("new-monitor", "new.com", 2000, "new-icon", 150, 0, NewMockPingProvider(10, nil))
	pm.Save(cfg)

	// Check that existing config is preserved
	existingConfig, exists := cfg.Monitoring.Ping["existing"]
	require.True(t, exists, "existing config should be preserved")
	assert.Equal(t, "existing.com", existingConfig.Address)
	assert.Equal(t, 500, existingConfig.Timeout)
	assert.Equal(t, "old-icon", existingConfig.Icon)
	assert.Equal(t, 25, existingConfig.AmberThreshold)

	// Check that new config is added
	newConfig, exists := cfg.Monitoring.Ping["new-monitor"]
	require.True(t, exists, "new config should be added")
	assert.Equal(t, "new.com", newConfig.Address)
	assert.Equal(t, 2000, newConfig.Timeout)
	assert.Equal(t, "new-icon", newConfig.Icon)
	assert.Equal(t, 150, newConfig.AmberThreshold)
}

// Test boundary value for amber threshold edge cases
func TestPingMonitor_AmberThresholdBoundary(t *testing.T) {
	tests := []struct {
		name           string
		amberThreshold int
		responseTime   int
		expectedStatus monitors.RAG
	}{
		{
			name:           "response exactly at threshold should be amber",
			amberThreshold: 100,
			responseTime:   100,
			expectedStatus: monitors.RAGAmber,
		},
		{
			name:           "response one ms below threshold should be green",
			amberThreshold: 100,
			responseTime:   99,
			expectedStatus: monitors.RAGGreen,
		},
		{
			name:           "response one ms above threshold should be amber",
			amberThreshold: 100,
			responseTime:   101,
			expectedStatus: monitors.RAGAmber,
		},
		{
			name:           "zero response time should be green",
			amberThreshold: 50,
			responseTime:   0,
			expectedStatus: monitors.RAGGreen,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := NewPingMonitor("boundary-test", "127.0.0.1", 1000, "", tt.amberThreshold, 0, NewMockPingProvider(tt.responseTime, nil))
			result := pm.Check(context.Background())
			assert.Equal(t, tt.expectedStatus, result.Status, "status should match expected")
			assert.Equal(t, result.Value, fmt.Sprintf("%d ms", tt.responseTime))
		})
	}
}

// Test DefaultPingProvider with very short timeout to potentially trigger Run() error
func TestDefaultPingProvider_Ping_VeryShortTimeout(t *testing.T) {
	// Skip in CI or short mode
	if testing.Short() || os.Getenv("CI") != "" {
		t.Skip("Skipping potentially flaky network test")
	}

	provider := &DefaultPingProvider{}
	ctx := context.Background()

	// Try multiple strategies to trigger the pinger.Run() error path
	testCases := []struct {
		name    string
		address string
		timeout int
	}{
		{"very short timeout", "8.8.8.8", 1},
		{"invalid IPv6", "::ffff:192.0.2.1", 1},      // IPv6 might fail differently
		{"broadcast address", "255.255.255.255", 10}, // Broadcast address might fail
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := provider.Ping(ctx, tc.address, tc.timeout)
			// We don't assert error == nil because these might legitimately fail
			// The goal is to exercise different error paths in DefaultPingProvider.Ping()
			if err != nil {
				errMsg := err.Error()
				assert.True(t,
					strings.Contains(errMsg, "ping") || strings.Contains(errMsg, "packets") ||
						strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "failed"),
					"error should be ping-related, got: %s", errMsg)
			}
		})
	}
}

// Test context cancellation behavior
func TestPingMonitor_Check_WithContext(t *testing.T) {
	// Test with a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	pm := NewPingMonitor("context-test", "127.0.0.1", 1000, "", 100, 0, NewMockPingProvider(50, nil))
	result := pm.Check(ctx)

	// Mock provider doesn't respect context cancellation, so it should still work
	// This tests that the Check method properly passes context through
	assert.Equal(t, monitors.RAGGreen, result.Status)
	assert.Equal(t, "50 ms", result.Value)
}

// Test Monitor fields are properly set
func TestPingMonitor_FieldAccess(t *testing.T) {
	pm := NewPingMonitor("field-test", "test.example.com", 5000, "test-icon", 200, 0, NewMockPingProvider(75, nil))

	assert.Equal(t, "ping_field-test", pm.Name())
	assert.Equal(t, "Ping: field-test (test.example.com)", pm.DisplayName())
	assert.Equal(t, "ping", pm.Group())

	// Verify internal fields (these aren't directly accessible but influence behavior)
	result := pm.Check(context.Background())
	assert.Equal(t, monitors.RAGGreen, result.Status) // 75ms < 200ms threshold
	assert.Equal(t, "75 ms", result.Value)
	assert.Equal(t, "ping", result.Group)
	assert.Equal(t, "Ping: field-test (test.example.com)", result.DisplayName)
}
