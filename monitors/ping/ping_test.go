package ping

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"lmon/config"
	"lmon/monitors"
)

func TestPingMonitor_SuccessGreen(t *testing.T) {
	pm := NewPingMonitor("test-green", "127.0.0.1", 1000, "", 100, &MockPingProvider{ResponseMs: 50, Err: nil})
	result := pm.Check(context.Background())
	if result.Status != monitors.RAGGreen {
		t.Errorf("Expected RAGGreen, got %v", result.Status)
	}
	if result.Value != "50 ms" {
		t.Errorf("Expected value '50 ms', got %s", result.Value)
	}
}

func TestPingMonitor_SuccessAmber(t *testing.T) {
	pm := NewPingMonitor("test-amber", "127.0.0.1", 1000, "", 100, &MockPingProvider{ResponseMs: 150, Err: nil})
	result := pm.Check(context.Background())
	if result.Status != monitors.RAGAmber {
		t.Errorf("Expected RAGAmber, got %v", result.Status)
	}
	if result.Value != "150 ms" {
		t.Errorf("Expected value '150 ms', got %s", result.Value)
	}
}

func TestPingMonitor_FailureRed(t *testing.T) {
	pm := NewPingMonitor("test-red", "127.0.0.1", 1000, "", 100, &MockPingProvider{Err: errors.New("timeout")})
	result := pm.Check(context.Background())
	if result.Status != monitors.RAGRed {
		t.Errorf("Expected RAGRed, got %v", result.Status)
	}
	if result.Value == "" || result.Value == "0 ms" {
		t.Errorf("Expected error message, got %s", result.Value)
	}
}

func TestPingMonitor_DisplayNameAndGroup(t *testing.T) {
	pm := NewPingMonitor("display", "localhost", 1000, "", 100, &MockPingProvider{ResponseMs: 10})
	if pm.DisplayName() != "Ping: display" {
		t.Errorf("Unexpected display name: %s", pm.DisplayName())
	}
	if pm.Group() != Group {
		t.Errorf("Unexpected group: %s", pm.Group())
	}
	if fmt.Sprintf("%s_%s", pm.Group(), pm.Name()) != "health_display" {
		t.Errorf("Unexpected name: %s", fmt.Sprintf("%s_%s", pm.Group(), pm.Name()))
	}
}

func TestPingMonitor_Save(t *testing.T) {
	cfg := &config.Config{}
	pm := NewPingMonitor("save-test", "1.2.3.4", 1234, "wifi", 200, &MockPingProvider{ResponseMs: 10})
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

func TestDefaultPingProvider_ParseOutput(t *testing.T) {
	// This test is now less relevant since pro-bing is used, but keep for completeness.
	mockOutput := "64 bytes from 1.2.3.4: icmp_seq=1 ttl=64 time=123 ms"
	lines := strings.Split(mockOutput, "\n")
	found := false
	for _, line := range lines {
		if strings.Contains(line, "time=") {
			parts := strings.Split(line, "time=")
			if len(parts) > 1 {
				timeStr := strings.Fields(parts[1])[0]
				var ms int
				_, err := fmt.Sscanf(timeStr, "%d", &ms)
				if err == nil && ms == 123 {
					found = true
				}
			}
		}
	}
	if !found {
		t.Errorf("Failed to parse ping output")
	}

	mockOutput = "no time here"
	lines = strings.Split(mockOutput, "\n")
	found = false
	for _, line := range lines {
		if strings.Contains(line, "time=") {
			found = true
		}
	}
	if found {
		t.Errorf("Should not find time= in output")
	}
}

func TestDefaultPingProvider_Ping_Error(t *testing.T) {
	provider := &DefaultPingProvider{}
	ctx := context.Background()
	_, err := provider.Ping(ctx, "invalid.invalid", 100)
	if err == nil {
		t.Errorf("Expected error for invalid ping address")
	}
}

func TestPingMonitor_Save_NilConfig(t *testing.T) {
	pm := NewPingMonitor("nil-test", "127.0.0.1", 1000, "", 100, &MockPingProvider{ResponseMs: 10})
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when saving to nil config")
		}
	}()
	pm.Save(nil)
}

func TestPingMonitor_FallbackElapsedTime(t *testing.T) {
	provider := &MockPingProvider{ResponseMs: 0, Err: nil}
	pm := NewPingMonitor("fallback-elapsed", "127.0.0.1", 1000, "", 100, provider)
	result := pm.Check(context.Background())
	if result.Value != "0 ms" {
		t.Errorf("Expected fallback value '0 ms', got %s", result.Value)
	}
}

func TestDefaultPingProvider_Ping_Localhost(t *testing.T) {
	provider := &DefaultPingProvider{}
	ctx := context.Background()
	ms, err := provider.Ping(ctx, "127.0.0.1", 1000)
	if err != nil {
		t.Fatalf("Ping to localhost failed: %v", err)
	}
	if ms < 0 {
		t.Errorf("Ping response time should be >= 0, got %d", ms)
	}
}

func TestDefaultPingProvider_Ping_Unreachable(t *testing.T) {
	provider := &DefaultPingProvider{}
	ctx := context.Background()
	// 203.0.113.0 is a TEST-NET-3 address, reserved for documentation and should not reply to ICMP echo
	_, err := provider.Ping(ctx, "203.0.113.0", 1000)
	if err == nil {
		t.Errorf("Expected error for unreachable TEST-NET-3 address")
	}
}
