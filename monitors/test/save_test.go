package monitors_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"lmon/config"
	"lmon/monitors"
	"lmon/monitors/disk"
	"lmon/monitors/healthcheck"
	"lmon/monitors/ping"
)

// TestService_Save verifies that Service.Save correctly persists all monitor types to config.
func TestService_Save(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	svc := monitors.NewService(ctx, time.Second, time.Second, nil)

	// Add a Disk monitor
	diskMon := disk.NewDisk("disk1", "/mnt/disk1", 75, "hdd", nil)
	svc.Add(ctx, diskMon)

	// Add a Healthcheck monitor
	healthMon, err := healthcheck.NewHealthcheck("health1", "http://localhost/health", 5, 401, "activity", "", nil, nil, nil)
	require.NoError(t, err)
	svc.Add(ctx, healthMon)

	// Add a Ping monitor
	pingMon := ping.NewPingMonitor("ping1", "127.0.0.1", 1000, "wifi", 100, nil)
	svc.Add(ctx, pingMon)

	cfg := &config.Config{}
	err = svc.Save(cfg)
	require.NoError(t, err, "service save should not error")

	// Check that each monitor type is present in config
	diskCfg, diskOk := cfg.Monitoring.Disk["disk1"]
	healthCfg, healthOk := cfg.Monitoring.Healthcheck["health1"]
	pingCfg, pingOk := cfg.Monitoring.Ping["ping1"]

	assert.True(t, diskOk, "disk monitor should be saved")
	assert.True(t, healthOk, "healthcheck monitor should be saved")
	assert.True(t, pingOk, "ping monitor should be saved")

	// Optionally, check some values
	assert.Equal(t, "/mnt/disk1", diskCfg.Path)
	assert.Equal(t, "hdd", diskCfg.Icon)
	assert.Equal(t, "http://localhost/health", healthCfg.URL)
	assert.Equal(t, 401, healthCfg.RespCode)
	assert.Equal(t, "activity", healthCfg.Icon)
	assert.Equal(t, "127.0.0.1", pingCfg.Address)
	assert.Equal(t, "wifi", pingCfg.Icon)
	assert.Equal(t, 100, pingCfg.AmberThreshold)
}
