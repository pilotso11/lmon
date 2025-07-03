package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Verify default values
	assert.Equal(t, "0.0.0.0", cfg.Web.Host)
	assert.Equal(t, 8080, cfg.Web.Port)
	assert.Equal(t, 60, cfg.Monitoring.Interval)
	assert.Equal(t, 1, len(cfg.Monitoring.Disk))
	assert.Equal(t, "/", cfg.Monitoring.Disk[0].Path)
	assert.Equal(t, 80, cfg.Monitoring.Disk[0].Threshold)
	assert.Equal(t, "storage", cfg.Monitoring.Disk[0].Icon)
	assert.Equal(t, 80, cfg.Monitoring.System.CPUThreshold)
	assert.Equal(t, 80, cfg.Monitoring.System.MemoryThreshold)
	assert.Equal(t, "memory", cfg.Monitoring.System.CPUIcon)
	assert.Equal(t, "memory", cfg.Monitoring.System.MemoryIcon)
	assert.Equal(t, 0, len(cfg.Monitoring.Healthchecks))
	assert.False(t, cfg.Webhook.Enabled)
	assert.Equal(t, "", cfg.Webhook.URL)
}

func TestSaveAndLoad(t *testing.T) {
	// Create a temporary file for testing
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	// Create a test configuration
	cfg := DefaultConfig()
	cfg.Web.Host = "127.0.0.1"
	cfg.Web.Port = 9090
	cfg.Monitoring.Interval = 30
	cfg.Monitoring.Disk[0].Threshold = 90
	cfg.Monitoring.System.CPUThreshold = 70
	cfg.Webhook.Enabled = true
	cfg.Webhook.URL = "https://example.com/webhook"

	// Add a health check
	cfg.Monitoring.Healthchecks = append(cfg.Monitoring.Healthchecks, HealthcheckConfig{
		Name:     "Test API",
		URL:      "https://api.example.com/health",
		Interval: 60,
		Timeout:  5,
		Icon:     "cloud",
	})

	// Save the configuration
	err = Save(cfg, tmpfile.Name())
	require.NoError(t, err)

	// Print the contents of the saved file for debugging
	data, err := os.ReadFile(tmpfile.Name())
	require.NoError(t, err)
	t.Logf("Saved file contents:\n%s", string(data))

	// Set up environment for Load
	os.Setenv("LMON_WEB_HOST", "")
	os.Setenv("LMON_WEB_PORT", "")

	// Load the configuration from the temporary file
	loadedCfg, err := LoadFromFile(tmpfile.Name())
	require.NoError(t, err)

	// Verify loaded values match saved values
	assert.Equal(t, cfg.Web.Host, loadedCfg.Web.Host, "Web.Host mismatch")
	assert.Equal(t, cfg.Web.Port, loadedCfg.Web.Port, "Web.Port mismatch")
	assert.Equal(t, cfg.Monitoring.Interval, loadedCfg.Monitoring.Interval, "Monitoring.Interval mismatch")
	assert.Equal(t, cfg.Monitoring.Disk[0].Threshold, loadedCfg.Monitoring.Disk[0].Threshold, "Disk[0].Threshold mismatch")
	assert.Equal(t, cfg.Monitoring.System.CPUThreshold, loadedCfg.Monitoring.System.CPUThreshold, "System.CPUThreshold mismatch")
	assert.Equal(t, cfg.Webhook.Enabled, loadedCfg.Webhook.Enabled, "Webhook.Enabled mismatch")
	assert.Equal(t, cfg.Webhook.URL, loadedCfg.Webhook.URL, "Webhook.URL mismatch")

	// Verify health check was loaded
	require.Equal(t, 1, len(loadedCfg.Monitoring.Healthchecks))
	assert.Equal(t, "Test API", loadedCfg.Monitoring.Healthchecks[0].Name)
	assert.Equal(t, "https://api.example.com/health", loadedCfg.Monitoring.Healthchecks[0].URL)
}

func TestLoadWithEnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("LMON_WEB_HOST", "192.168.1.1")
	os.Setenv("LMON_WEB_PORT", "8888")
	os.Setenv("LMON_MONITORING_INTERVAL", "45")
	os.Setenv("LMON_WEBHOOK_ENABLED", "true")
	os.Setenv("LMON_WEBHOOK_URL", "https://example.org/webhook")
	defer func() {
		os.Unsetenv("LMON_WEB_HOST")
		os.Unsetenv("LMON_WEB_PORT")
		os.Unsetenv("LMON_MONITORING_INTERVAL")
		os.Unsetenv("LMON_WEBHOOK_ENABLED")
		os.Unsetenv("LMON_WEBHOOK_URL")
	}()

	// Load configuration
	cfg, err := Load()
	require.NoError(t, err)

	// Verify environment variables were applied
	assert.Equal(t, "192.168.1.1", cfg.Web.Host)
	assert.Equal(t, 8888, cfg.Web.Port)
	assert.Equal(t, 45, cfg.Monitoring.Interval)
	assert.True(t, cfg.Webhook.Enabled)
	assert.Equal(t, "https://example.org/webhook", cfg.Webhook.URL)
}
